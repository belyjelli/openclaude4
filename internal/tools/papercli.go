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
	paperCLIDefaultSec = 120
	paperCLIMaxSec     = 600
)

// PaperCLIRegistered reports whether the PaperCLI tool would be registered in [NewDefaultRegistry]
// (papercli on PATH, or OPENCLAUDE_PAPERCLI / PAPERCLI_BIN set).
func PaperCLIRegistered() bool {
	_, ok := paperCLIResolvePath()
	return ok
}

func paperCLIResolvePath() (string, bool) {
	for _, k := range []string{"OPENCLAUDE_PAPERCLI", "PAPERCLI_BIN"} {
		if p := strings.TrimSpace(os.Getenv(k)); p != "" {
			return os.ExpandEnv(p), true
		}
	}
	p, err := exec.LookPath("papercli")
	return p, err == nil
}

func registerPaperCLIIfAvailable(r *Registry) {
	if !PaperCLIRegistered() {
		return
	}
	r.Register(PaperCLI{})
}

// PaperCLI runs the external papercli binary (OCR, scrape, split/join, wallet helpers, Arkham, etc.).
// See https://github.com/morpheum-labs/purewalletcli and local docs in the papercli repo.
// Registered when papercli is on PATH, or when OPENCLAUDE_PAPERCLI / PAPERCLI_BIN points to the binary.
type PaperCLI struct{}

func (PaperCLI) Name() string      { return "PaperCLI" }
func (PaperCLI) IsDangerous() bool { return true }

func (PaperCLI) Description() string {
	return "Run the **papercli** tool (agent-oriented CLI: OCR image→text, scrape/normalize secrets from messy text, split/join large files, wallet/mnemonic utilities, Arkham API calls). " +
		"Pass argv as the arguments after `papercli` (e.g. [\"version\"], [\"ocr\", \"./scan.png\"]). " +
		"Requires papercli on PATH, or set OPENCLAUDE_PAPERCLI or PAPERCLI_BIN to the binary. " +
		"Uses workspace-relative cwd; configure papercli via PAPERCLI_CONFIG or ~/.papercli/config.json per upstream docs."
}

func (PaperCLI) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"argv": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Arguments for papercli (not including the papercli executable name). May be empty to print top-level help.",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory relative to workspace (default: workspace root)",
			},
			"timeout_seconds": map[string]any{
				"type":        "number",
				"description": fmt.Sprintf("Timeout in seconds (default %d, max %d)", paperCLIDefaultSec, paperCLIMaxSec),
			},
		},
		"required": []string{"argv"},
	}
}

func argvFromArgs(args map[string]any) ([]string, error) {
	raw, ok := args["argv"]
	if !ok {
		return nil, fmt.Errorf("argv is required")
	}
	switch x := raw.(type) {
	case []string:
		return x, nil
	case []any:
		out := make([]string, 0, len(x))
		for i, e := range x {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("argv[%d] must be a string", i)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("argv must be an array of strings")
	}
}

func (PaperCLI) Execute(ctx context.Context, args map[string]any) (string, error) {
	argv, err := argvFromArgs(args)
	if err != nil {
		return "", err
	}

	bin, ok := paperCLIResolvePath()
	if !ok {
		return "", fmt.Errorf("papercli not found (install to PATH or set OPENCLAUDE_PAPERCLI)")
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

	sec := float64(paperCLIDefaultSec)
	if v, ok := args["timeout_seconds"].(float64); ok && v > 0 {
		sec = v
	}
	if sec > paperCLIMaxSec {
		sec = paperCLIMaxSec
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
			return "", fmt.Errorf("papercli: %w (%s)", runErr, msg)
		}
		return "", fmt.Errorf("papercli: %w", runErr)
	}

	if !utf8.Valid(out) {
		return "", fmt.Errorf("papercli stdout is not valid UTF-8")
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
		return "(papercli produced no stdout)", nil
	}
	return text + truncNote, nil
}
