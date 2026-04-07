package tui

import (
	"context"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gitlawb/openclaude4/internal/core"
)

// Run starts the Bubble Tea UI, wiring the agent to kernel events only (stdout from the model is discarded).
func Run(cfg Config) error {
	if cfg.Ctx == nil {
		cfg.Ctx = context.Background()
	}
	pb := newPermBridge(cfg.Ctx, cfg.AutoApprove)

	var agent *core.Agent
	cfg.Registry.Register(core.NewTaskTool(func() *core.Agent { return agent }))

	m := newModel(cfg, nil, func() *core.Agent { return agent }, pb)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(cfg.Ctx))
	pb.setProgram(p)
	m.send = p.Send

	agent = &core.Agent{
		Client:   cfg.Client,
		Registry: cfg.Registry,
		Out:      io.Discard,
		Confirm:  pb.Confirm,
		OnEvent: func(e core.Event) {
			p.Send(kernelMsg{e: e})
		},
	}

	_, err := p.Run()
	return err
}
