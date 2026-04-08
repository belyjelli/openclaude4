package tui

import (
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// ThemeHolder stores the active TUI palette mode (thread-safe).
type ThemeHolder struct {
	mu sync.Mutex
	v  string
}

// NewThemeHolder returns a holder with mode "auto".
func NewThemeHolder() *ThemeHolder {
	return &ThemeHolder{v: "auto"}
}

// Set updates the theme mode: light, dark, or auto.
func (h *ThemeHolder) Set(mode string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		m = "auto"
	}
	h.v = m
}

// Get returns light, dark, or auto.
func (h *ThemeHolder) Get() string {
	if h == nil {
		return "auto"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.v == "" {
		return "auto"
	}
	return h.v
}

// MarkdownStyle returns a glamour profile name ("light" or "dark").
func (h *ThemeHolder) MarkdownStyle() string {
	switch h.Get() {
	case "light":
		return "light"
	case "dark":
		return "dark"
	default:
		if lipgloss.HasDarkBackground() {
			return "dark"
		}
		return "light"
	}
}
