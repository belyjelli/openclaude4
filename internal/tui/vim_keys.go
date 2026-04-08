package tui

import (
	"sync/atomic"
)

// VimKeysHolder toggles vim-style single-line prompt editing (TUI only).
type VimKeysHolder struct {
	on atomic.Bool
}

// NewVimKeysHolder returns a holder with vim-style keys off.
func NewVimKeysHolder() *VimKeysHolder {
	return &VimKeysHolder{}
}

// Enabled reports whether vim-style keybindings are active.
func (h *VimKeysHolder) Enabled() bool {
	if h == nil {
		return false
	}
	return h.on.Load()
}

// Set turns vim-style keys on or off.
func (h *VimKeysHolder) Set(on bool) {
	if h == nil {
		return
	}
	h.on.Store(on)
}

// Toggle flips vim-style keys and returns a one-line status for the transcript.
func (h *VimKeysHolder) Toggle() string {
	if h == nil {
		return ""
	}
	next := !h.on.Load()
	h.on.Store(next)
	if next {
		return "vim-style prompt keys on — Esc → normal · i/I/a/A → insert · hjkl · 0$^ · x · Enter sends line (/vim again to off)"
	}
	return "vim-style prompt keys off (default line editing)"
}
