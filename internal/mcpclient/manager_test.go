package mcpclient

import (
	"context"
	"testing"

	"github.com/gitlawb/openclaude4/internal/tools"
)

func TestConnectAndRegister_empty(t *testing.T) {
	reg := tools.NewRegistry()
	m := ConnectAndRegister(context.Background(), reg, nil, nil)
	if m == nil {
		t.Fatal("expected non-nil Manager")
	}
	m.Close()
}

func TestManager_Close_nil(t *testing.T) {
	var m *Manager
	m.Close()
}
