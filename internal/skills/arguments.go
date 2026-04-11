package skills

import (
	"fmt"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// ParseArguments splits a slash-command argument tail using bash-like rules
// (comments with #, quotes, etc.) for positional placeholders.
func ParseArguments(args string) []string {
	s := strings.TrimSpace(args)
	if s == "" {
		return nil
	}
	words, err := parseArgsWithShellParser(s)
	if err != nil || len(words) == 0 {
		return strings.Fields(s)
	}
	return words
}

func parseArgsWithShellParser(line string) ([]string, error) {
	r := strings.NewReader(strings.TrimSpace(line) + "\n")
	p := syntax.NewParser()
	file, err := p.Parse(r, "")
	if err != nil {
		return nil, err
	}
	if len(file.Stmts) == 0 {
		return nil, fmt.Errorf("empty")
	}
	var out []string
	for _, st := range file.Stmts {
		call, ok := st.Cmd.(*syntax.CallExpr)
		if !ok {
			continue
		}
		for _, arg := range call.Args {
			w, err := wordLiteral(arg)
			if err != nil {
				continue
			}
			if w != "" {
				out = append(out, w)
			}
		}
	}
	return out, nil
}

func wordLiteral(p *syntax.Word) (string, error) {
	if p == nil || len(p.Parts) == 0 {
		return "", fmt.Errorf("empty word")
	}
	var b strings.Builder
	for _, part := range p.Parts {
		switch x := part.(type) {
		case *syntax.Lit:
			b.WriteString(x.Value)
		case *syntax.SglQuoted:
			b.WriteString(x.Value)
		case *syntax.DblQuoted:
			for _, qp := range x.Parts {
				switch q := qp.(type) {
				case *syntax.Lit:
					b.WriteString(q.Value)
				default:
					return "", fmt.Errorf("unsupported in dbl quoted")
				}
			}
		default:
			return "", fmt.Errorf("unsupported part")
		}
	}
	return b.String(), nil
}

// SubstituteArguments replaces v3-style placeholders in content (see openclaude3 argumentSubstitution.ts).
func SubstituteArguments(content string, args string, appendIfNoPlaceholder bool, argumentNames []string) string {
	parsed := ParseArguments(args)
	original := content

	for i, name := range argumentNames {
		if name == "" {
			continue
		}
		val := ""
		if i < len(parsed) {
			val = parsed[i]
		}
		content = replaceNamedArgument(content, name, val)
	}

	content = replaceARGUMENTSBracket(content, parsed)
	content = replaceDollarDigits(content, parsed)
	content = strings.ReplaceAll(content, "$ARGUMENTS", args)

	if content == original && appendIfNoPlaceholder && strings.TrimSpace(args) != "" {
		content = content + "\n\nARGUMENTS: " + args
	}
	return content
}

func replaceNamedArgument(s, name, val string) string {
	prefix := "$" + name
	var b strings.Builder
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], prefix)
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		j += i
		b.WriteString(s[i:j])
		next := j + len(prefix)
		if next < len(s) {
			c := s[next]
			if c == '[' || isWordChar(c) {
				b.WriteString(s[j : j+1]) // literal $ only; continue search after it
				i = j + 1
				continue
			}
		}
		b.WriteString(val)
		i = next
	}
	return b.String()
}

func replaceARGUMENTSBracket(s string, parsed []string) string {
	const prefix = "$ARGUMENTS["
	for {
		i := strings.Index(s, prefix)
		if i < 0 {
			return s
		}
		close := strings.IndexByte(s[i+len(prefix):], ']')
		if close < 0 {
			return s
		}
		close += i + len(prefix)
		numStr := s[i+len(prefix) : close]
		idx, err := strconv.Atoi(numStr)
		if err != nil {
			s = s[:i] + s[close+1:]
			continue
		}
		val := ""
		if idx >= 0 && idx < len(parsed) {
			val = parsed[idx]
		}
		s = s[:i] + val + s[close+1:]
	}
}

func replaceDollarDigits(s string, parsed []string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '$' || i+1 >= len(s) || s[i+1] < '0' || s[i+1] > '9' {
			b.WriteByte(s[i])
			continue
		}
		j := i + 1
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j < len(s) && isWordChar(s[j]) {
			b.WriteByte(s[i])
			continue
		}
		idx, err := strconv.Atoi(s[i+1 : j])
		if err != nil {
			b.WriteString(s[i:j])
			i = j - 1
			continue
		}
		if idx >= 0 && idx < len(parsed) {
			b.WriteString(parsed[idx])
		}
		i = j - 1
	}
	return b.String()
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}
