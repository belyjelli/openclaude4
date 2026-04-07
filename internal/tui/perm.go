package tui

import (
	"context"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type permBridge struct {
	mu          sync.Mutex
	prog        *tea.Program
	ctx         context.Context
	autoApprove bool
}

func newPermBridge(ctx context.Context, autoApprove bool) *permBridge {
	return &permBridge{ctx: ctx, autoApprove: autoApprove}
}

func (b *permBridge) setProgram(p *tea.Program) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.prog = p
}

func (b *permBridge) Confirm(toolName string, args map[string]any) bool {
	if b.autoApprove {
		return true
	}
	b.mu.Lock()
	p := b.prog
	b.mu.Unlock()
	if p == nil {
		return false
	}
	ch := make(chan bool, 1)
	p.Send(permPromptMsg{tool: toolName, args: args, result: ch})
	select {
	case <-b.ctx.Done():
		return false
	case v, ok := <-ch:
		if !ok {
			return false
		}
		return v
	}
}

type permPromptMsg struct {
	tool   string
	args   map[string]any
	result chan bool
}
