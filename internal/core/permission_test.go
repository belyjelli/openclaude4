package core

import (
	"strings"
	"testing"
)

func TestDeclineToolMessage(t *testing.T) {
	if s := DeclineToolMessage(""); s != "User declined this tool execution." {
		t.Fatalf("empty: %q", s)
	}
	if s := DeclineToolMessage("  use smaller diff  "); !strings.Contains(s, "use smaller diff") {
		t.Fatalf("with note: %q", s)
	}
}
