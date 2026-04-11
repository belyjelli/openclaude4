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

	rawConfirm := pb.Confirm
	agent = &core.Agent{
		Client:   cfg.Client,
		Registry: cfg.Registry,
		Out:      io.Discard,
		OnEvent: func(e core.Event) {
			p.Send(kernelMsg{e: e})
		},
	}
	if cfg.PermissionEngine != nil {
		eng := cfg.PermissionEngine
		agent.PermissionPolicy = func(n string, a map[string]any) (core.PermissionOutcome, bool, string) {
			return eng.Eval(n, a)
		}
	}
	agent.Confirm = func(n string, a map[string]any) core.PermissionOutcome {
		o := rawConfirm(n, a)
		if cfg.PermissionStore != nil && len(o.AddAllowRules) > 0 {
			_ = cfg.PermissionStore.AppendAllow(o.AddAllowRules)
		}
		if cfg.PermissionEngine != nil && len(o.AddAllowRules) > 0 {
			cfg.PermissionEngine.AppendAllow(o.AddAllowRules)
		}
		if o.EnableSessionAutoApprove && cfg.AutoApprove != nil {
			cfg.AutoApprove.Store(true)
		}
		return o
	}
	if cfg.Live != nil {
		cfg.Live.BindAgent(agent)
	}
	if cfg.Theme != nil {
		ApplyTheme(cfg.Theme.Get())
	}

	_, err := p.Run()
	return err
}
