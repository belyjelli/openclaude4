// Package chatlive holds mutable session state for REPL/TUI (e.g. swapping API client after /model).
package chatlive

import (
	"sync"

	"github.com/gitlawb/openclaude4/internal/core"
)

// LiveChat tracks the active stream client and keeps Agent.Client in sync when swapped.
type LiveChat struct {
	mu     sync.Mutex
	client core.StreamClient
	agent  *core.Agent
}

// New returns a LiveChat with the initial client.
func New(client core.StreamClient) *LiveChat {
	return &LiveChat{client: client}
}

// Client returns the current stream client.
func (l *LiveChat) Client() core.StreamClient {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.client
}

// BindAgent wires the agent so [SwapClient] updates Agent.Client.
func (l *LiveChat) BindAgent(a *core.Agent) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.agent = a
}

// SwapClient replaces the live client and assigns it to the bound agent (if any).
func (l *LiveChat) SwapClient(c core.StreamClient) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.client = c
	if l.agent != nil {
		l.agent.Client = c
	}
}
