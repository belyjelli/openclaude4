package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/core"
)

func TestHandleProviderWizard_TUI_StartsProviderWizard(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	st := chatState{providerWizardIn: nil, allowConfigEditorWizard: true}
	err := handleProviderWizard(st, &buf)
	var want core.SlashStartProviderWizard
	if !errors.As(err, &want) {
		t.Fatalf("expected SlashStartProviderWizard, got %v", err)
	}
}

func TestHandleProviderWizard_NoStdin_StaticGuide(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	st := chatState{client: nil, providerWizardIn: nil}
	if err := handleProviderWizard(st, &buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "plain REPL") || !strings.Contains(s, "Copy-paste reference") {
		t.Fatalf("expected static guide when stdin wizard unavailable, got:\n%s", s)
	}
}

func TestRunProviderInteractiveWizard_Cancel(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var buf bytes.Buffer
	if err := runProviderInteractiveWizard(chatState{}, &buf, in, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "cancelled") {
		t.Fatalf("%q", buf.String())
	}
}

func TestRunProviderInteractiveWizard_OpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	in := strings.NewReader("1\n1\n\n")
	var buf bytes.Buffer
	if err := runProviderInteractiveWizard(chatState{}, &buf, in, nil); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, `name: openai`) || !strings.Contains(s, "gpt-4o-mini") {
		t.Fatalf("missing openai yaml: %q", s)
	}
}

func TestRunProviderInteractiveWizard_BackFromOpenAI(t *testing.T) {
	t.Parallel()
	// openai → model step → back to menu → empty cancel
	in := strings.NewReader("1\nb\n\n")
	var buf bytes.Buffer
	if err := runProviderInteractiveWizard(chatState{}, &buf, in, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "cancelled") {
		t.Fatalf("expected cancel after back to root: %q", buf.String())
	}
}

func TestRunProviderInteractiveWizard_GitHub(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("4\n1\n1\n")
	var buf bytes.Buffer
	if err := runProviderInteractiveWizard(chatState{}, &buf, in, nil); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, `name: github`) || !strings.Contains(s, "gpt-4o") {
		t.Fatalf("missing github yaml: %q", s)
	}
}

func TestRunProviderInteractiveWizard_OpenRouter(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("5\n1\n\n")
	var buf bytes.Buffer
	if err := runProviderInteractiveWizard(chatState{}, &buf, in, nil); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, `name: openrouter`) || !strings.Contains(s, "openai/gpt-4o-mini") {
		t.Fatalf("missing openrouter yaml: %q", s)
	}
}
