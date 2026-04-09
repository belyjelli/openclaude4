package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	speedtestCLIDefaultSec = 300
	speedtestCLIMaxSec     = 900
)

// SpeedtestCLIRegistered reports whether the SpeedtestCLI tool would be registered in [NewDefaultRegistry]
// (librespeed-cli or speedtcli on PATH, or OPENCLAUDE_SPEEDTEST_CLI / SPEEDTEST_CLI_BIN set).
func SpeedtestCLIRegistered() bool {
	_, ok := speedtestCLIResolvePath()
	return ok
}

// SpeedtestCLIBinary returns the resolved executable path when [SpeedtestCLIRegistered] is true.
func SpeedtestCLIBinary() (string, bool) {
	return speedtestCLIResolvePath()
}

func speedtestCLIResolvePath() (string, bool) {
	for _, k := range []string{"OPENCLAUDE_SPEEDTEST_CLI", "SPEEDTEST_CLI_BIN"} {
		if p := strings.TrimSpace(os.Getenv(k)); p != "" {
			return os.ExpandEnv(p), true
		}
	}
	for _, name := range []string{"librespeed-cli", "speedtcli"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, true
		}
	}
	return "", false
}

func registerSpeedtestCLIIfAvailable(r *Registry) {
	if !SpeedtestCLIRegistered() {
		return
	}
	r.Register(SpeedtestCLI{})
}

// SpeedtestCLI runs the external LibreSpeed CLI (ping, jitter, download, upload; --json / --list / --server, etc.).
// Registered when librespeed-cli or speedtcli is on PATH, or OPENCLAUDE_SPEEDTEST_CLI / SPEEDTEST_CLI_BIN points to the binary
// (e.g. a local build from https://github.com/librespeed/speedtest-cli).
type SpeedtestCLI struct{}

func (SpeedtestCLI) Name() string      { return "SpeedtestCLI" }
func (SpeedtestCLI) IsDangerous() bool { return true }

func (SpeedtestCLI) Description() string {
	return "Run the **LibreSpeed** command-line speed test (network bandwidth/latency; optional --json, --list, --server, --simple). " +
		"Pass argv as arguments after the binary (e.g. [\"--version\"], [\"--json\"], [\"--list\"]). " +
		"Requires **librespeed-cli** or **speedtcli** on PATH, or set **OPENCLAUDE_SPEEDTEST_CLI** or **SPEEDTEST_CLI_BIN** to the binary. " +
		"Uses workspace-relative cwd when cwd is set."
}

func (SpeedtestCLI) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"argv": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Arguments for the speedtest CLI (not including the executable name). May be empty (default run).",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory relative to workspace (default: workspace root)",
			},
			"timeout_seconds": map[string]any{
				"type":        "number",
				"description": fmt.Sprintf("Timeout in seconds (default %d, max %d)", speedtestCLIDefaultSec, speedtestCLIMaxSec),
			},
		},
		"required": []string{"argv"},
	}
}

func (SpeedtestCLI) Execute(ctx context.Context, args map[string]any) (string, error) {
	argv, err := argvFromArgs(args)
	if err != nil {
		return "", err
	}

	bin, ok := speedtestCLIResolvePath()
	if !ok {
		return "", fmt.Errorf("speedtest CLI not found (install librespeed-cli or speedtcli to PATH, or set OPENCLAUDE_SPEEDTEST_CLI)")
	}

	absRoot, err := resolveUnderWorkdir(ctx, ".")
	if err != nil {
		return "", err
	}
	cwd := absRoot
	if c, ok := args["cwd"].(string); ok && strings.TrimSpace(c) != "" {
		cwd, err = resolveUnderWorkdir(ctx, c)
		if err != nil {
			return "", err
		}
	}

	sec := float64(speedtestCLIDefaultSec)
	if v, ok := args["timeout_seconds"].(float64); ok && v > 0 {
		sec = v
	}
	if sec > speedtestCLIMaxSec {
		sec = speedtestCLIMaxSec
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(sec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, bin, argv...)
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	out := stdout.Bytes()
	errText := strings.TrimSpace(stderr.String())
	if runErr != nil {
		msg := strings.TrimSpace(errText)
		if msg != "" {
			return "", fmt.Errorf("speedtest-cli: %w (%s)", runErr, msg)
		}
		return "", fmt.Errorf("speedtest-cli: %w", runErr)
	}

	if !utf8.Valid(out) {
		return "", fmt.Errorf("speedtest-cli stdout is not valid UTF-8")
	}
	text := strings.TrimSpace(string(out))
	if errText != "" {
		if text != "" {
			text += "\n\n--- stderr ---\n" + errText
		} else {
			text = errText
		}
	}
	truncNote := ""
	if maxChars := webFetchMaxCharsLimit; maxChars > 0 && utf8.RuneCountInString(text) > maxChars {
		runes := []rune(text)
		text = string(runes[:maxChars])
		truncNote = fmt.Sprintf("\n\n[truncated to %d characters]", maxChars)
	}
	if text == "" {
		return "(speedtest-cli produced no stdout)", nil
	}
	return text + truncNote, nil
}
