package bashv2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gitlawb/openclaude4/internal/bashv2/audit"
	"github.com/gitlawb/openclaude4/internal/bashv2/output"
	"github.com/gitlawb/openclaude4/internal/bashv2/parse"
	"github.com/gitlawb/openclaude4/internal/bashv2/snapshot"
	"github.com/gitlawb/openclaude4/internal/bashv2/validate"
)

// PolicyHook evaluates allow/deny rules for a tool (typically "Bash"). When decided is false,
// no rule matched and the caller continues to safe-readonly patterns / ask.
type PolicyHook func(toolName string, args map[string]any) (decided bool, allow bool, reason string)

// SessionOpts configures a long-lived Bash execution session (snapshot, policy, callbacks).
type SessionOpts struct {
	Config                Config
	Policy                PolicyHook
	SafeReadOnlyNoConfirm func(string) bool
	// OnOutputChunk is optional streaming hook (toolCallID, chunk, runningTotalBytes).
	OnOutputChunk func(toolCallID, chunk string, totalBytes int)
}

// Session is created once per chat / agent run and reused for every Bash tool call.
type Session struct {
	opts SessionOpts
	mu   sync.Mutex
	snap *snapshot.Data
	sink *audit.Sink
}

// NewSession constructs a session; snapshot is captured lazily on first Gate/Execute.
func NewSession(opts SessionOpts) *Session {
	s := &Session{opts: opts, sink: audit.NewSink(opts.Config.AuditLogPath)}
	return s
}

func (s *Session) ensureSnapshot(ctx context.Context, workspace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snap != nil {
		return nil
	}
	if workspace == "" {
		return fmt.Errorf("bashv2: workspace root is empty")
	}
	shell := os.Getenv("SHELL")
	snap, err := snapshot.CaptureShellState(ctx, workspace, shell)
	if err != nil {
		return err
	}
	s.snap = snap
	return nil
}

func commandHash(cmd string) string {
	sum := sha256.Sum256([]byte(cmd))
	return hex.EncodeToString(sum[:12])
}

// Gate runs L1–L4 (canonicalize, parse, validate, policy) without spawning a shell.
func (s *Session) Gate(ctx context.Context, workspace string, args map[string]any) GateResult {
	cmd, _ := args["command"].(string)
	cmd = strings.TrimSpace(CanonicalizeCommand(cmd))
	if cmd == "" {
		return GateResult{Phase: PhaseDeny, Reason: "empty_command", Message: "command is required"}
	}
	if err := s.ensureSnapshot(ctx, workspace); err != nil {
		return GateResult{Phase: PhaseDeny, Reason: "snapshot_error", Message: err.Error()}
	}

	units, err := parse.SplitUnits(cmd)
	if err != nil {
		return GateResult{Phase: PhaseDeny, Reason: "parse_error", Message: err.Error()}
	}
	if verdict, id, reason := validate.Chain(validate.DefaultChain(), units); verdict == validate.Fail {
		return GateResult{Phase: PhaseDeny, Reason: id, Message: reason}
	}

	argsForPolicy := maps.Clone(args)
	argsForPolicy["command"] = cmd

	if s.opts.Policy != nil {
		if ok, allow, tag := s.opts.Policy("Bash", argsForPolicy); ok {
			if !allow {
				if s.sink != nil {
					s.sink.Log(audit.Entry{Phase: "gate", Reason: tag, Workspace: workspace, CommandHash: commandHash(cmd)})
				}
				return GateResult{Phase: PhaseDeny, Reason: tag, Message: "Permission policy denied this Bash command."}
			}
			if s.sink != nil {
				s.sink.Log(audit.Entry{Phase: "gate", Reason: tag, Workspace: workspace, CommandHash: commandHash(cmd)})
			}
			return GateResult{Phase: PhaseAllow, Reason: tag}
		}
	}
	if s.opts.SafeReadOnlyNoConfirm != nil && s.opts.SafeReadOnlyNoConfirm(cmd) {
		if s.sink != nil {
			s.sink.Log(audit.Entry{Phase: "gate", Reason: "safe_readonly_pattern", Workspace: workspace, CommandHash: commandHash(cmd)})
		}
		return GateResult{Phase: PhaseAllow, Reason: "safe_readonly_pattern"}
	}
	if s.sink != nil {
		s.sink.Log(audit.Entry{Phase: "gate", Reason: "needs_confirmation", Workspace: workspace, CommandHash: commandHash(cmd)})
	}
	return GateResult{Phase: PhaseAsk, Reason: "needs_confirmation"}
}

// ExecuteInput carries runtime execution parameters (post Gate/Confirm).
type ExecuteInput struct {
	ToolCallID string
	Args       map[string]any
	Workspace  string // absolute workspace root
}

// Execute runs L6–L7: build runner, spawn (sandboxed when configured), post-process output.
func (s *Session) Execute(ctx context.Context, in ExecuteInput) (string, error) {
	cmdStr, _ := in.Args["command"].(string)
	cmdStr = strings.TrimSpace(CanonicalizeCommand(cmdStr))
	if cmdStr == "" {
		return "", fmt.Errorf("command is required")
	}
	if err := s.ensureSnapshot(ctx, in.Workspace); err != nil {
		return "", err
	}

	units, err := parse.SplitUnits(cmdStr)
	if err != nil {
		return "", err
	}
	if verdict, id, reason := validate.Chain(validate.DefaultChain(), units); verdict == validate.Fail {
		return "", fmt.Errorf("%s: %s", id, reason)
	}

	cwd := in.Workspace
	if c, ok := in.Args["cwd"].(string); ok && strings.TrimSpace(c) != "" {
		c = filepath.Clean(c)
		if c == ".." || strings.HasPrefix(c, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("cwd escapes workspace")
		}
		cwd = filepath.Join(in.Workspace, c)
	}
	if rel, err := filepath.Rel(in.Workspace, cwd); err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("cwd outside workspace")
	}

	sec := 120.0
	if v, ok := in.Args["timeout_seconds"].(float64); ok && v > 0 {
		sec = v
	}
	maxSec := s.opts.Config.MaxTimeoutSeconds
	if maxSec <= 0 {
		maxSec = 600
	}
	if sec > maxSec {
		sec = maxSec
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(float64(time.Second)*sec))
	defer cancel()

	bodyPath, err := snapshot.WriteBodyScript(in.Workspace, "bash-body", cmdStr)
	if err != nil {
		return "", err
	}
	runnerPath, err := snapshot.WriteRunnerScript(in.Workspace, s.snap.MaterializedPath, bodyPath, cwd)
	if err != nil {
		return "", err
	}

	env := snapshotExecEnv()
	runCmd, err := s.prepareRunnerCommand(execCtx, runnerPath, cwd, in.Workspace, env)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	runCmd.Stdout = &buf
	runCmd.Stderr = &buf
	streamHook := StreamHookFromContext(ctx)
	if streamHook == nil {
		streamHook = s.opts.OnOutputChunk
	}
	if streamHook != nil && in.ToolCallID != "" {
		pr, pw := io.Pipe()
		defer func() { _ = pw.Close() }()
		runCmd.Stdout = io.MultiWriter(&buf, pw)
		runCmd.Stderr = io.MultiWriter(&buf, pw)
		go streamChunks(in.ToolCallID, pr, streamHook)
	}

	runErr := runCmd.Run()
	exit := 0
	if runErr != nil {
		if x, ok := runErr.(*exec.ExitError); ok && x.ProcessState != nil {
			exit = x.ExitCode()
		} else {
			return strings.TrimSpace(buf.String()), fmt.Errorf("%w: %s", runErr, strings.TrimSpace(buf.String()))
		}
	}

	combined := buf.Bytes()
	out, persisted, perr := output.PostProcess(in.Workspace, cmdStr, combined, s.opts.Config.InlineOutputMaxBytes)
	if perr != nil {
		return "", perr
	}
	if !output.SemanticSuccess(cmdStr, exit) && exit != 0 {
		if out != "" {
			return strings.TrimSpace(out), fmt.Errorf("exit %d: %s", exit, strings.TrimSpace(out))
		}
		return "", fmt.Errorf("exit %d", exit)
	}

	if s.sink != nil {
		s.sink.Log(audit.Entry{
			Phase:         "execute",
			Reason:        "ok",
			SnapshotVer:   s.snap.Version,
			Workspace:     in.Workspace,
			CWD:           cwd,
			CommandHash:   commandHash(cmdStr),
			ExitCode:      exit,
			OutputBytes:   len(combined),
			PersistedPath: persisted,
			Sandbox:       s.sandboxLabel(),
			ToolCallID:    in.ToolCallID,
		})
	}
	return strings.TrimSpace(out), nil
}

func streamChunks(toolCallID string, r *io.PipeReader, emit func(string, string, int)) {
	defer func() { _ = r.Close() }()
	total := 0
	b := make([]byte, 4096)
	for {
		n, err := r.Read(b)
		if n > 0 {
			total += n
			emit(toolCallID, string(b[:n]), total)
		}
		if err != nil {
			return
		}
	}
}

func snapshotExecEnv() []string {
	// Small deterministic env; snapshot script re-exports captured vars.
	var out []string
	for _, k := range []string{"PATH", "HOME", "USER", "LANG", "LC_ALL", "LC_CTYPE", "TERM", "TMPDIR"} {
		if v := os.Getenv(k); v != "" {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func (s *Session) sandboxLabel() string {
	if s.opts.Config.SandboxDisabled {
		return "disabled"
	}
	if runtime.GOOS == "linux" && s.opts.Config.LinuxUseBubblewrap {
		if _, err := exec.LookPath("bwrap"); err == nil {
			return "linux_bwrap"
		}
		return "linux_direct"
	}
	if runtime.GOOS == "darwin" {
		return "darwin_" + strings.ToLower(strings.TrimSpace(s.opts.Config.DarwinSandbox))
	}
	return runtime.GOOS + "_direct"
}

func (s *Session) prepareRunnerCommand(ctx context.Context, runnerPath, cwd, workspace string, env []string) (*exec.Cmd, error) {
	cfg := s.opts.Config
	if cfg.SandboxDisabled {
		cmd := exec.CommandContext(ctx, "bash", runnerPath)
		configureCmd(cmd, cwd, env)
		return cmd, nil
	}
	if runtime.GOOS == "linux" && cfg.LinuxUseBubblewrap {
		bp, err := exec.LookPath("bwrap")
		if err == nil {
			return linuxBwrapCommand(ctx, bp, runnerPath, cwd, workspace, env)
		}
		if cfg.StrictLinuxSandbox {
			return nil, fmt.Errorf("bubblewrap (bwrap) is required for Bash on Linux but was not found on PATH")
		}
	}
	if runtime.GOOS == "darwin" && strings.EqualFold(strings.TrimSpace(cfg.DarwinSandbox), "required") {
		return nil, fmt.Errorf("bashv2: darwinSandbox=required is not yet supported (no Seatbelt profile); use best_effort or off")
	}
	cmd := exec.CommandContext(ctx, "bash", runnerPath)
	configureCmd(cmd, cwd, env)
	return cmd, nil
}

func configureCmd(cmd *exec.Cmd, cwd string, env []string) {
	cmd.Dir = cwd
	cmd.Env = env
}

func linuxBwrapCommand(ctx context.Context, bwrapPath, runnerPath, cwd, workspace string, env []string) (*exec.Cmd, error) {
	absRunner, err := filepath.Abs(runnerPath)
	if err != nil {
		return nil, err
	}
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	absWs, err := filepath.Abs(workspace)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(absCwd, absWs) {
		return nil, fmt.Errorf("cwd must be inside workspace for sandbox")
	}

	args := []string{
		bwrapPath,
		"--die-with-parent",
		"--new-session",
		"--proc", "/proc",
		"--dev-bind", "/dev", "/dev",
		"--ro-bind", "/", "/",
		"--bind", absWs, absWs,
		"--chdir", absCwd,
	}
	if _, err := os.Stat("/etc/resolv.conf"); err == nil {
		args = append(args, "--ro-bind", "/etc/resolv.conf", "/etc/resolv.conf")
	}
	tmp := os.TempDir()
	if tmp != "" {
		if absTmp, err := filepath.Abs(tmp); err == nil {
			args = append(args, "--bind", absTmp, absTmp)
			args = append(args, "--setenv", "TMPDIR="+absTmp)
		}
	}
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") || strings.HasPrefix(e, "HOME=") || strings.HasPrefix(e, "TMPDIR=") {
			args = append(args, "--setenv", e)
		}
	}
	bashPath, err := exec.LookPath("bash")
	if err != nil || bashPath == "" {
		bashPath = "/bin/bash"
	}
	args = append(args, bashPath, absRunner)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	configureCmd(cmd, absCwd, nil)
	return cmd, nil
}

