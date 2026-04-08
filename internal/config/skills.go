package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// SkillDirs returns directories to scan for skills (see [github.com/gitlawb/openclaude4/internal/skills]).
// Order: config skills.dirs, OPENCLAUDE_SKILLS_DIRS (comma-separated), then default ./.openclaude/skills
// and ~/.local/share/openclaude/skills when those paths exist (defaults are skipped if missing).
func SkillDirs() []string {
	var out []string
	if v := viper.GetStringSlice("skills.dirs"); len(v) > 0 {
		for _, p := range v {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	if e := strings.TrimSpace(os.Getenv("OPENCLAUDE_SKILLS_DIRS")); e != "" {
		for _, p := range strings.Split(e, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	if cwd != "" {
		def := filepath.Join(cwd, ".openclaude", "skills")
		if dirExists(def) {
			out = append(out, def)
		}
	}
	if home != "" {
		def := filepath.Join(home, ".local", "share", "openclaude", "skills")
		if dirExists(def) {
			out = append(out, def)
		}
	}
	return dedupeStrPreserveOrder(out)
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func dedupeStrPreserveOrder(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
