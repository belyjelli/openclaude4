package tools

import "strings"

// IterLimitPickCandidates lists tool names for iteration-limit retry UIs.
// Task is excluded (scoped retry omits it). If slashAllow is non-empty, only those names are included.
func IterLimitPickCandidates(reg *Registry, slashAllow []string) []string {
	if reg == nil {
		return nil
	}
	names := reg.SortedNames()
	allowSet := make(map[string]struct{})
	for _, a := range slashAllow {
		a = strings.TrimSpace(a)
		if a != "" {
			allowSet[a] = struct{}{}
		}
	}
	var out []string
	for _, n := range names {
		if n == "Task" {
			continue
		}
		if len(allowSet) > 0 {
			if _, ok := allowSet[n]; !ok {
				continue
			}
		}
		out = append(out, n)
	}
	return out
}
