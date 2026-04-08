package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gitlawb/openclaude4/internal/session"
	sdk "github.com/sashabaranov/go-openai"
)

const maxInlineExportBytes = 512 * 1024

func slashExport(st chatState, args []string, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if st.messages == nil || len(*st.messages) == 0 {
		_, _ = fmt.Fprintln(out, "(no messages to export)")
		return nil
	}

	format := "json"
	path := ""

	switch len(args) {
	case 0:
	case 1:
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "json":
			// stdout json
		case "md", "markdown":
			format = "md"
		default:
			path = args[0]
		}
	default:
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "json":
			path = strings.TrimSpace(strings.Join(args[1:], " "))
		case "md", "markdown":
			format = "md"
			path = strings.TrimSpace(strings.Join(args[1:], " "))
		default:
			return fmt.Errorf("usage: /export [json|md] [path]  or  /export <path.json> (json to file)")
		}
	}

	if path != "" {
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			return fmt.Errorf("export path %q is a directory", path)
		}
	}

	msgs := session.RepairTranscript(*st.messages)
	wd, _ := os.Getwd()
	id := "inline-export"
	if st.persist != nil && st.persist.store != nil && strings.TrimSpace(st.persist.store.ID) != "" {
		id = st.persist.store.ID
	}
	snap := session.FileV1{
		Version:   1,
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		WorkDir:   wd,
		Messages:  msgs,
	}

	if format == "md" {
		body := exportTranscriptMarkdown(msgs)
		if path != "" {
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(out, "(exported markdown to %s)\n", path)
			return nil
		}
		if len(body) > maxInlineExportBytes {
			return fmt.Errorf("export is larger than %d bytes — use: /export md <path>", maxInlineExportBytes)
		}
		_, _ = fmt.Fprint(out, body)
		if !strings.HasSuffix(body, "\n") {
			_, _ = fmt.Fprintln(out)
		}
		return nil
	}

	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	if path != "" {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir for export: %w", err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "(exported JSON to %s)\n", path)
		return nil
	}
	if len(raw) > maxInlineExportBytes {
		return fmt.Errorf("export is larger than %d bytes — use: /export json <path>", maxInlineExportBytes)
	}
	_, _ = out.Write(raw)
	_, _ = fmt.Fprintln(out)
	return nil
}

func exportTranscriptMarkdown(msgs []sdk.ChatCompletionMessage) string {
	var b strings.Builder
	for _, m := range msgs {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		switch role {
		case "system":
			_, _ = fmt.Fprintf(&b, "## system\n\n%s\n\n", strings.TrimSpace(messageContentForExport(m)))
		case "user":
			_, _ = fmt.Fprintf(&b, "## user\n\n%s\n\n", strings.TrimSpace(messageContentForExport(m)))
		case "assistant":
			line := strings.TrimSpace(messageContentForExport(m))
			if len(m.ToolCalls) > 0 {
				var names []string
				for _, tc := range m.ToolCalls {
					names = append(names, tc.Function.Name)
				}
				if line != "" {
					line += "\n\n"
				}
				line += "(tool_calls: " + strings.Join(names, ", ") + ")"
			}
			_, _ = fmt.Fprintf(&b, "## assistant\n\n%s\n\n", line)
		case "tool":
			_, _ = fmt.Fprintf(&b, "## tool (%s)\n\n%s\n\n", strings.TrimSpace(m.Name), strings.TrimSpace(m.Content))
		default:
			_, _ = fmt.Fprintf(&b, "## %s\n\n%s\n\n", role, strings.TrimSpace(messageContentForExport(m)))
		}
	}
	return b.String()
}

func messageContentForExport(m sdk.ChatCompletionMessage) string {
	if strings.TrimSpace(m.Content) != "" {
		return m.Content
	}
	var parts []string
	for _, p := range m.MultiContent {
		if p.Type == sdk.ChatMessagePartTypeText && strings.TrimSpace(p.Text) != "" {
			parts = append(parts, p.Text)
		}
	}
	return strings.Join(parts, "\n")
}
