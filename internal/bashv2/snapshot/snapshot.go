package snapshot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Data is a frozen shell environment fragment used for reproducible Bash runs.
type Data struct {
	Version   string // content hash
	ShellPath string
	// MaterializedPath is the host path to a shell script that recreates env (source this first).
	MaterializedPath string
}

// CaptureShellState probes the user's shell for export/alias/function state and writes
// a reproducible snapshot file under workDir/.openclaude/tmp.
func CaptureShellState(ctx context.Context, workDir, shell string) (*Data, error) {
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = "/bin/bash"
	}
	tmpDir := filepath.Join(workDir, ".openclaude", "tmp")
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return nil, err
	}

	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	// Best-effort: capture exports and a bounded alias/function dump.
	script := `set +o posix 2>/dev/null || true
export -p 2>/dev/null || env | sed -n 's/^/export /p'
echo "# --- aliases ---"
alias -p 2>/dev/null || true
echo "# --- functions (truncated) ---"
declare -f 2>/dev/null | head -c 200000 || true
`
	cmd := exec.CommandContext(probeCtx, shell, "-c", script)
	cmd.Dir = workDir
	cmd.Env = minimalEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Fall back to PATH-only snapshot.
		out = []byte(fmt.Sprintf("# snapshot fallback (probe failed: %v)\nexport PATH=%q\n", err, os.Getenv("PATH")))
	}

	sum := sha256.Sum256(out)
	version := hex.EncodeToString(sum[:8])
	name := "bash-snapshot-" + version + ".sh"
	path := filepath.Join(tmpDir, name)
	// Owner read/write so the same snapshot hash can be refreshed on a later session in the same workspace.
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return nil, err
	}
	return &Data{
		Version:          version,
		ShellPath:        shell,
		MaterializedPath: path,
	}, nil
}

func minimalEnv() []string {
	var keep []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		switch strings.ToLower(k) {
		case "path", "home", "user", "lang", "lc_all", "lc_ctype", "term", "tmpdir":
			keep = append(keep, kv)
		}
	}
	if !hasKeyPrefix(keep, "HOME=") {
		if h, err := os.UserHomeDir(); err == nil {
			keep = append(keep, "HOME="+h)
		}
	}
	if !hasKeyPrefix(keep, "PATH=") {
		keep = append(keep, "PATH="+os.Getenv("PATH"))
	}
	return keep
}

func hasKeyPrefix(env []string, prefix string) bool {
	for _, e := range env {
		if strings.HasPrefix(strings.ToLower(e), strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

// WriteBodyScript writes the user command body to an isolated script file.
func WriteBodyScript(workDir, namePrefix, body string) (string, error) {
	tmpDir := filepath.Join(workDir, ".openclaude", "tmp")
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(body))
	f := filepath.Join(tmpDir, fmt.Sprintf("%s-%s.sh", namePrefix, hex.EncodeToString(sum[:6])))
	if err := os.WriteFile(f, []byte("#!/usr/bin/env bash\nset -euo pipefail\n"+body+"\n"), 0o500); err != nil {
		return "", err
	}
	return f, nil
}

// WriteRunnerScript writes the host-controlled runner that sources the snapshot then execs the body.
func WriteRunnerScript(workDir, snapshotPath, bodyScriptPath, cwd string) (string, error) {
	tmpDir := filepath.Join(workDir, ".openclaude", "tmp")
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return "", err
	}
	var b bytes.Buffer
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -euo pipefail\n")
	b.WriteString("set +o histexpand 2>/dev/null || true\n")
	b.WriteString("set -o pipefail 2>/dev/null || true\n")
	b.WriteString(fmt.Sprintf("IFS=$' \\t\\n'\n"))
	b.WriteString(fmt.Sprintf("source %q\n", snapshotPath))
	b.WriteString(fmt.Sprintf("cd %q\n", cwd))
	b.WriteString(fmt.Sprintf("exec bash %q\n", bodyScriptPath))
	sum := sha256.Sum256(b.Bytes())
	name := "bash-runner-" + hex.EncodeToString(sum[:8]) + ".sh"
	path := filepath.Join(tmpDir, name)
	if err := os.WriteFile(path, b.Bytes(), 0o500); err != nil {
		return "", err
	}
	return path, nil
}
