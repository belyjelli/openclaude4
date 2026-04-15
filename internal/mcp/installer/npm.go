package installer

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
)

// NPM implements NPMDetector from the MCP v2 installer spec.
type NPM struct{}

func (NPM) Name() string              { return "NPMDetector" }
func (NPM) ConfidenceWeight() float64 { return 0.9 }

// PackageJSON is the subset of package.json used for MCP detection.
type PackageJSON struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Keywords    []string          `json:"keywords"`
	Bin         json.RawMessage   `json:"bin"`
	Scripts     map[string]string `json:"scripts"`
	MCP         map[string]any    `json:"mcp"`
}

var (
	npxPattern        = regexp.MustCompile(`npx\s+-y\s+(@[\w.-]+/[\w.-]+|[\w@./-]+)(?:\s+([^\n]+))?`)
	bunxPattern       = regexp.MustCompile(`bunx\s+-y\s+(@[\w.-]+/[\w.-]+|[\w@./-]+)(?:\s+([^\n]+))?`)
	genericNPMPattern = regexp.MustCompile(`(?:npx|bunx)\s+-y\s+([\w@./-]+)(?:\s+([^\n]+))?`)
)

func hasBunInstalled() bool {
	_, err := exec.LookPath("bun")
	return err == nil
}

func (NPM) Detect(_ context.Context, files map[string][]byte, meta *RepoMetadata) ([]*Candidate, error) {
	if meta == nil {
		return nil, nil
	}
	pjKey := pickKey(files, "package.json")
	pjRaw := files[pjKey]
	readme := readmeBytes(files)

	triggered := len(pjRaw) > 0 && shouldInspectPackageJSON(pjRaw)
	if !triggered && len(readme) > 0 {
		if genericNPMPattern.Match(readme) {
			triggered = true
		}
	}
	if !triggered {
		return nil, nil
	}

	var pkg PackageJSON
	if len(pjRaw) > 0 {
		if err := json.Unmarshal(pjRaw, &pkg); err != nil {
			return nil, nil
		}
	}
	pkgName := strings.TrimSpace(pkg.Name)
	if pkgName == "" && len(readme) > 0 {
		if m := npxPattern.FindStringSubmatch(string(readme)); len(m) > 1 {
			pkgName = strings.TrimSpace(m[1])
		} else if m := bunxPattern.FindStringSubmatch(string(readme)); len(m) > 1 {
			pkgName = strings.TrimSpace(m[1])
		}
	}
	if pkgName == "" {
		return nil, nil
	}

	argsFromReadme := extractArgsFromReadme(readme, pkgName)
	hasBunLock := len(files["bun.lockb"]) > 0 || len(files["bunfig.toml"]) > 0
	readmeLower := strings.ToLower(string(readme))
	prefersBunx := hasBunLock || strings.Contains(readmeLower, "bunx")
	hasExactNpx := npxPattern.Match(readme)
	hasExactBunx := bunxPattern.Match(readme)

	confBase := npmBaseConfidence(pkgName, pkg)
	if keywordsContainMCP(pkg.Keywords) {
		confBase += 15
	}
	if strings.Contains(strings.ToLower(pkg.Description), "mcp server") {
		confBase += 10
	}

	buildNpx := func() *Candidate {
		cmd := []string{"npx", "-y", pkgName}
		if len(argsFromReadme) > 0 {
			cmd = append(cmd, argsFromReadme...)
		}
		conf := confBase
		if hasExactNpx {
			conf += 30
		}
		if strings.Contains(readmeLower, "npx") && !hasExactBunx {
			conf += 5
		}
		if conf > 100 {
			conf = 100
		}
		return &Candidate{
			Name:         sanitizeServerName(pkgName, meta.Repo),
			Transport:    "stdio",
			Command:      cmd,
			Approval:     "ask",
			Confidence:   conf,
			Reason:       "NPM package " + pkgName + " (npx)",
			DetectedFrom: "NPMDetector",
		}
	}

	buildBunx := func() *Candidate {
		cmd := []string{"bunx", "-y", pkgName}
		if len(argsFromReadme) > 0 {
			cmd = append(cmd, argsFromReadme...)
		}
		conf := confBase
		if hasExactBunx {
			conf += 30
		}
		if prefersBunx {
			conf += 10
		}
		if hasBunInstalled() {
			conf += 15
		}
		if conf > 100 {
			conf = 100
		}
		return &Candidate{
			Name:         sanitizeServerName(pkgName, meta.Repo),
			Transport:    "stdio",
			Command:      cmd,
			Approval:     "ask",
			Confidence:   conf,
			Reason:       "NPM package " + pkgName + " (bunx)",
			DetectedFrom: "NPMDetector",
		}
	}

	var out []*Candidate
	out = append(out, buildNpx())
	if hasBunInstalled() {
		out = append(out, buildBunx())
	} else if prefersBunx {
		b := buildBunx()
		b.Confidence = maxFloat(0, b.Confidence-25)
		b.Reason += " (bun not in PATH)"
		out = append(out, b)
	}

	return out, nil
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func pickKey(files map[string][]byte, base string) string {
	for k := range files {
		if strings.HasSuffix(strings.ToLower(k), strings.ToLower(base)) {
			return k
		}
	}
	return base
}

func readmeBytes(files map[string][]byte) []byte {
	for _, name := range []string{"README.md", "README.markdown", "readme.md"} {
		if b, ok := files[name]; ok && len(b) > 0 {
			return b
		}
		for k, v := range files {
			if strings.EqualFold(k, name) {
				return v
			}
		}
	}
	return nil
}

func shouldInspectPackageJSON(raw []byte) bool {
	var probe struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Keywords    []string `json:"keywords"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	n := strings.ToLower(probe.Name)
	d := strings.ToLower(probe.Description)
	if strings.HasPrefix(n, "@modelcontextprotocol/server-") {
		return true
	}
	if matched, _ := regexp.MatchString(`^@[\w-]+/mcp-server-`, n); matched {
		return true
	}
	if strings.HasPrefix(n, "mcp-server-") {
		return true
	}
	if strings.Contains(n, "mcp") || strings.Contains(d, "mcp") {
		return true
	}
	for _, kw := range probe.Keywords {
		if strings.EqualFold(strings.TrimSpace(kw), "mcp") {
			return true
		}
	}
	return false
}

func isOfficialScope(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), "@modelcontextprotocol/")
}

func keywordsContainMCP(keywords []string) bool {
	for _, k := range keywords {
		if strings.EqualFold(strings.TrimSpace(k), "mcp") {
			return true
		}
	}
	return false
}

func npmBaseConfidence(pkgName string, pkg PackageJSON) float64 {
	var c float64
	if isOfficialScope(pkgName) {
		c += 40
	}
	if strings.Contains(strings.ToLower(pkgName), "mcp-server") {
		c += 25
	}
	if strings.Contains(strings.ToLower(pkgName), "mcp") {
		c += 10
	}
	return c
}

func extractArgsFromReadme(readme []byte, pkgName string) []string {
	s := string(readme)
	for _, re := range []*regexp.Regexp{npxPattern, bunxPattern, genericNPMPattern} {
		for _, m := range re.FindAllStringSubmatch(s, -1) {
			if len(m) < 2 {
				continue
			}
			matched := strings.TrimSpace(m[1])
			if matched != pkgName && !strings.HasSuffix(matched, pkgName) && !strings.Contains(matched, pkgName) {
				continue
			}
			if len(m) >= 3 {
				rest := strings.TrimSpace(m[2])
				if rest == "" {
					return nil
				}
				return strings.Fields(rest)
			}
			return nil
		}
	}
	return nil
}

func sanitizeServerName(pkgName, repoFallback string) string {
	base := pkgName
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	base = strings.TrimPrefix(base, "@")
	base = strings.ToLower(base)
	re := regexp.MustCompile(`[^a-z0-9_-]+`)
	base = re.ReplaceAllString(base, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		base = strings.ToLower(repoFallback)
		base = re.ReplaceAllString(base, "_")
	}
	if base == "" {
		base = "mcp_server"
	}
	return base
}
