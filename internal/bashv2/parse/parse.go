package parse

import (
	"fmt"
	"strings"
)

// MaxUnits is the hard cap on parsed segments per invocation (anti-DoS).
const MaxUnits = 50

// SplitUnits splits command on top-level `;`, `&&`, and `||` while respecting
// single-quoted, double-quoted, and parentheses depth for subshells (best-effort).
func SplitUnits(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("empty command")
	}
	var units []string
	var b strings.Builder
	paren := 0
	var quote byte // 0, '\'', '"'
	esc := false
	flush := func() {
		s := strings.TrimSpace(b.String())
		b.Reset()
		if s != "" {
			units = append(units, s)
		}
	}
	i := 0
	for i < len(command) {
		c := command[i]
		if quote == '"' {
			if esc {
				b.WriteByte(c)
				esc = false
				i++
				continue
			}
			if c == '\\' {
				esc = true
				i++
				continue
			}
			if c == '"' {
				quote = 0
				b.WriteByte(c)
				i++
				continue
			}
			b.WriteByte(c)
			i++
			continue
		}
		if quote == '\'' {
			if c == '\'' {
				quote = 0
			}
			b.WriteByte(c)
			i++
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
			b.WriteByte(c)
			i++
			continue
		case '(':
			paren++
			b.WriteByte(c)
			i++
			continue
		case ')':
			if paren > 0 {
				paren--
			}
			b.WriteByte(c)
			i++
			continue
		}
		if paren == 0 {
			if c == ';' {
				flush()
				i++
				continue
			}
			if strings.HasPrefix(command[i:], "&&") {
				flush()
				i += 2
				continue
			}
			if strings.HasPrefix(command[i:], "||") {
				flush()
				i += 2
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in command")
	}
	flush()
	if len(units) == 0 {
		return nil, fmt.Errorf("empty command after parse")
	}
	if len(units) > MaxUnits {
		return nil, fmt.Errorf("command exceeds maximum of %d segments (%d found)", MaxUnits, len(units))
	}
	return units, nil
}
