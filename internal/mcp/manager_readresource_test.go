package mcp

import (
	"context"
	"strings"
	"testing"
)

func TestManager_ReadResourceText_nil(t *testing.T) {
	var m *Manager
	_, err := m.ReadResourceText(context.Background(), "srv", "uri:x")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "nil") {
		t.Fatalf("got %v", err)
	}
}
