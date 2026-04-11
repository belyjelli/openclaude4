package mentions

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SyntheticAtPath is a structured file reference (e.g. from an IDE over gRPC).
type SyntheticAtPath struct {
	Path      string // absolute or relative to working directory
	LineStart int32  // 1-based; 0 = omit
	LineEnd   int32  // 1-based inclusive; 0 with LineStart>0 means single line
}

func formatLineSuffix(lineStart, lineEnd int32) string {
	if lineStart <= 0 {
		return ""
	}
	if lineEnd > 0 && lineEnd != lineStart {
		return fmt.Sprintf("#L%d-%d", lineStart, lineEnd)
	}
	return fmt.Sprintf("#L%d", lineStart)
}

// AppendSyntheticAtMentions appends @-path tokens (v3-style) derived from structured IDE paths.
// wd should be the absolute workspace directory used to shorten absolute paths.
func AppendSyntheticAtMentions(userText string, wd string, paths []SyntheticAtPath) string {
	if len(paths) == 0 {
		return userText
	}
	wd = strings.TrimSpace(wd)
	var parts []string
	for _, p := range paths {
		disp := strings.TrimSpace(p.Path)
		if disp == "" {
			continue
		}
		if filepath.IsAbs(disp) && wd != "" {
			if awd, err := filepath.Abs(wd); err == nil {
				if ad, err := filepath.Abs(disp); err == nil {
					if r, err := filepath.Rel(awd, ad); err == nil {
						disp = r
					}
				}
			}
		}
		suf := formatLineSuffix(p.LineStart, p.LineEnd)
		var tok string
		if strings.ContainsAny(disp, " \t") {
			tok = `@"` + disp + `"` + suf
		} else {
			tok = "@" + disp + suf
		}
		parts = append(parts, tok)
	}
	if len(parts) == 0 {
		return userText
	}
	synth := strings.Join(parts, " ")
	ut := strings.TrimRight(userText, "\r\n")
	if ut == "" {
		return synth
	}
	return ut + " " + synth
}
