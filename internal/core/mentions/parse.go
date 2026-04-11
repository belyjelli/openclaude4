package mentions

import (
	"regexp"
	"strconv"
	"strings"
)

// FileSpec is one @-file mention after parsing #L line suffixes.
type FileSpec struct {
	Path       string
	LineStart  int // 1-based; 0 = whole file
	LineEnd    int // 1-based inclusive; 0 with LineStart>0 means single line
	RawToken   string // original @-token (for dedup/debug), may be empty
	WasQuoted  bool
}

// MCPResourceSpec is a v3-style @server:resourceURI mention (first segment is server name).
type MCPResourceSpec struct {
	Server string
	URI    string // full resource URI passed to MCP resources/read
}

var (
	quotedAtRe = regexp.MustCompile(`(?:^|\s)@"([^"]+)"`)
	// @ then non-space, not starting with " (quoted handled separately)
	regularAtRe = regexp.MustCompile(`(?:^|\s)@([^\s"]+)`)
	// v3: (^|\s)@(?!")([^\s"]+:[^\s"]+)\b
	mcpAtRe = regexp.MustCompile(`(?:^|\s)@(?!")([^\s"]+:[^\s"]+)\b`)
)

// ExtractMCPResourceMentions returns "server:uri" keys (no leading @), v3-compatible.
func ExtractMCPResourceMentions(content string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, m := range mcpAtRe.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		key := strings.TrimSpace(m[1])
		if key == "" {
			continue
		}
		// Windows drive: never MCP
		if matched, _ := regexp.MatchString(`^[A-Za-z]:[\\/]`, key); matched {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// ParseAtMentionedFileLines splits "file.txt#L10-20" into path and 1-based line range.
func ParseAtMentionedFileLines(mention string) (path string, lineStart, lineEnd int) {
	mention = strings.TrimSpace(mention)
	// ^([^#]+)(?:#L(\d+)(?:-(\d+))?)?(?:#[^#]*)?$
	re := regexp.MustCompile(`^([^#]+)(?:#L(\d+)(?:-(\d+))?)?(?:#[^#]*)?$`)
	sm := re.FindStringSubmatch(mention)
	if sm == nil {
		return mention, 0, 0
	}
	path = strings.TrimSpace(sm[1])
	if path == "" {
		path = mention
	}
	if len(sm) > 2 && sm[2] != "" {
		lineStart, _ = strconv.Atoi(sm[2])
	}
	if len(sm) > 3 && sm[3] != "" {
		lineEnd, _ = strconv.Atoi(sm[3])
	} else if lineStart > 0 {
		lineEnd = lineStart
	}
	return path, lineStart, lineEnd
}

func isDeferredAgentMention(pathPart string) bool {
	p := strings.TrimSpace(pathPart)
	// @agent-legacy and @"x (agent)" content
	if strings.HasPrefix(p, "agent-") {
		return true
	}
	if strings.HasSuffix(p, " (agent)") {
		return true
	}
	return false
}

// ExtractFileSpecs returns file @-mentions excluding MCP, @mcp:, and deferred @agent forms.
func ExtractFileSpecs(content string) []FileSpec {
	mcpKeys := map[string]struct{}{}
	for _, k := range ExtractMCPResourceMentions(content) {
		mcpKeys[k] = struct{}{}
	}

	seen := map[string]struct{}{}
	var specs []FileSpec

	add := func(rawWithAt, inner string, quoted bool) {
		inner = strings.TrimSpace(inner)
		if inner == "" {
			return
		}
		if strings.HasPrefix(strings.ToLower(inner), "mcp:") {
			return
		}
		if _, ok := mcpKeys[inner]; ok {
			return
		}
		pathPart, ls, le := ParseAtMentionedFileLines(inner)
		if isDeferredAgentMention(pathPart) || isDeferredAgentMention(inner) {
			return
		}
		dedup := pathPart + "|" + strconv.Itoa(ls) + "|" + strconv.Itoa(le) + "|" + strconv.FormatBool(quoted)
		if _, ok := seen[dedup]; ok {
			return
		}
		seen[dedup] = struct{}{}
		specs = append(specs, FileSpec{
			Path:      pathPart,
			LineStart: ls,
			LineEnd:   le,
			RawToken:  strings.TrimSpace(rawWithAt),
			WasQuoted: quoted,
		})
	}

	for _, m := range quotedAtRe.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		full := strings.TrimSpace(m[0])
		atIdx := strings.IndexByte(full, '@')
		raw := full[atIdx:]
		add(raw, m[1], true)
	}

	for _, m := range regularAtRe.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		tok := m[1]
		if strings.HasPrefix(tok, `"`) {
			continue
		}
		full := strings.TrimSpace(m[0])
		atIdx := strings.IndexByte(full, '@')
		raw := full[atIdx:]
		add(raw, tok, false)
	}

	return specs
}

// ParseMCPKeys splits "server:uriRest" into server and full URI for ReadResource.
func ParseMCPKey(key string) (server, uri string, ok bool) {
	key = strings.TrimSpace(key)
	i := strings.IndexByte(key, ':')
	if i <= 0 || i >= len(key)-1 {
		return "", "", false
	}
	if matched, _ := regexp.MatchString(`^[A-Za-z]:[\\/]`, key); matched {
		return "", "", false
	}
	return key[:i], key[i+1:], true
}
