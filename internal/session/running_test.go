package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterRunningListRunning(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	meta := RunningMeta{
		PID:       os.Getpid(),
		SessionID: "test-sess",
		CWD:       dir,
		Started:   "",
		TUI:       false,
		Provider:  "openai",
		Model:     "gpt-test",
	}
	cleanup, err := RegisterRunning(dir, meta)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)

	ents, err := ListRunning(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 {
		t.Fatalf("want 1 entry, got %d", len(ents))
	}
	if !ents[0].Alive {
		t.Fatal("expected current pid alive")
	}
	if ents[0].Meta.SessionID != "test-sess" {
		t.Fatalf("session id: got %q", ents[0].Meta.SessionID)
	}
	p := filepath.Join(dir, "running", "nonexistent.json")
	if err := os.WriteFile(p, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	ents2, err := ListRunning(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents2) != 1 {
		t.Fatalf("malformed json should be skipped: got %d entries", len(ents2))
	}
}
