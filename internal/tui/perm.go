package tui

import (
	"context"
	"sync"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gitlawb/openclaude4/internal/core"
)

type permBridge struct {
	mu   sync.Mutex
	prog *tea.Program
	ctx  context.Context
	// autoApprove is read on every Confirm; toggled at runtime in the TUI (Shift+Tab).
	autoApprove *atomic.Bool
}

func newPermBridge(ctx context.Context, autoApprove *atomic.Bool) *permBridge {
	if autoApprove == nil {
		v := new(atomic.Bool)
		autoApprove = v
	}
	return &permBridge{ctx: ctx, autoApprove: autoApprove}
}

func (b *permBridge) setProgram(p *tea.Program) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.prog = p
}

// Confirm blocks until the TUI resolves the permission dialog (or context cancelled).
func (b *permBridge) Confirm(toolName string, args map[string]any) core.PermissionOutcome {
	if b.autoApprove.Load() {
		return core.AllowPermission()
	}
	b.mu.Lock()
	p := b.prog
	b.mu.Unlock()
	if p == nil {
		return core.DenyPermission("")
	}
	ch := make(chan core.PermissionOutcome, 1)
	p.Send(permPromptMsg{tool: toolName, args: args, result: ch})
	select {
	case <-b.ctx.Done():
		return core.DenyPermission("")
	case v, ok := <-ch:
		if !ok {
			return core.DenyPermission("")
		}
		return v
	}
}

type permPromptMsg struct {
	tool   string
	args   map[string]any
	result chan core.PermissionOutcome
}
