package mentions

import (
	"context"
	"fmt"
	"os"
	pathpkg "path/filepath"
	"strings"

	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/tools"
)

// MaxTotalInjectBytes caps the combined size of appended attachment sections (excluding the original user line).
const MaxTotalInjectBytes = 768 * 1024

const maxDirEntries = 1000

// Deps carries optional services for mention expansion.
type Deps struct {
	MCP *mcpclient.Manager
}

// ExpandUserText appends fenced attachment sections for @files and @server:uri MCP mentions.
// The original userText is preserved first; expansion uses [tools.WithWorkDir] on ctx.
// @agent-* / teammate @name are not expanded (see package comment in doc.go).
func ExpandUserText(ctx context.Context, userText string, deps Deps) (string, error) {
	userText = strings.TrimRight(userText, "\r\n")
	specs := ExtractFileSpecs(userText)
	mcpKeys := ExtractMCPResourceMentions(userText)

	var sections []string
	total := 0

	for _, fs := range specs {
		sec, n, err := expandFileSpec(ctx, fs)
		if err != nil {
			return "", err
		}
		if err := budgetAdd(&total, n); err != nil {
			return "", err
		}
		sections = append(sections, sec)
	}

	for _, key := range mcpKeys {
		if deps.MCP == nil {
			return "", fmt.Errorf("MCP mention %q requires a connected MCP manager", key)
		}
		server, uri, ok := ParseMCPKey(key)
		if !ok {
			continue
		}
		sec, n, err := expandMCPKey(ctx, deps.MCP, server, uri, key)
		if err != nil {
			return "", err
		}
		if err := budgetAdd(&total, n); err != nil {
			return "", err
		}
		sections = append(sections, sec)
	}

	if len(sections) == 0 {
		return userText, nil
	}
	var b strings.Builder
	b.WriteString(userText)
	if !strings.HasSuffix(userText, "\n") && userText != "" {
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(strings.Join(sections, "\n"))
	return b.String(), nil
}

func budgetAdd(total *int, add int) error {
	if add <= 0 {
		return nil
	}
	if *total+add > MaxTotalInjectBytes {
		return fmt.Errorf("@-mention attachments exceed max size (%d bytes)", MaxTotalInjectBytes)
	}
	*total += add
	return nil
}

func expandFileSpec(ctx context.Context, fs FileSpec) (section string, byteLen int, err error) {
	rel := fs.Path
	abs, err := tools.ResolveUnderWorkdir(ctx, rel)
	if err != nil {
		return "", 0, fmt.Errorf("@%s: %w", rel, err)
	}
	root := tools.WorkDir(ctx)
	if root == "" {
		root, _ = os.Getwd()
	}
	display := rel
	if root != "" {
		if r, e := pathpkg.Rel(root, abs); e == nil {
			display = r
		}
	}

	fi, err := os.Stat(abs)
	if err != nil {
		return "", 0, fmt.Errorf("@%s: %w", rel, err)
	}

	var body string
	if fi.IsDir() {
		entries, err := os.ReadDir(abs)
		if err != nil {
			return "", 0, fmt.Errorf("@%s: %w", rel, err)
		}
		trunc := false
		if len(entries) > maxDirEntries {
			entries = entries[:maxDirEntries]
			trunc = true
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		body = strings.Join(names, "\n")
		if trunc {
			body += fmt.Sprintf("\n… and more entries (showing first %d)", maxDirEntries)
		}
	} else {
		raw, err := tools.ReadWorkspaceText(ctx, rel)
		if err != nil {
			return "", 0, fmt.Errorf("@%s: %w", rel, err)
		}
		if fs.LineStart > 0 {
			body = sliceLines(raw, fs.LineStart, fs.LineEnd)
			if body == "" && raw != "" {
				return "", 0, fmt.Errorf("@%s: line range L%d-%d is out of range", rel, fs.LineStart, fs.LineEnd)
			}
		} else {
			body = raw
		}
	}

	title := display
	if fs.LineStart > 0 {
		if fs.LineEnd > 0 && fs.LineEnd != fs.LineStart {
			title += fmt.Sprintf("#L%d-%d", fs.LineStart, fs.LineEnd)
		} else {
			title += fmt.Sprintf("#L%d", fs.LineStart)
		}
	}
	sec := "### Attached: " + title + "\n```\n" + body + "\n```"
	return sec, len(sec), nil
}

func sliceLines(text string, start, end int) string {
	if start < 1 || end < 1 {
		return text
	}
	if end < start {
		end = start
	}
	lines := strings.Split(text, "\n")
	if start > len(lines) {
		return ""
	}
	if end > len(lines) {
		end = len(lines)
	}
	chunk := lines[start-1 : end]
	return strings.Join(chunk, "\n")
}

func expandMCPKey(ctx context.Context, m *mcpclient.Manager, server, uri, rawKey string) (section string, byteLen int, err error) {
	text, err := m.ReadResourceText(ctx, server, uri)
	if err != nil {
		return "", 0, fmt.Errorf("@%s: %w", rawKey, err)
	}
	title := rawKey
	sec := "### Attached MCP resource: " + title + "\n```\n" + text + "\n```"
	return sec, len(sec), nil
}
