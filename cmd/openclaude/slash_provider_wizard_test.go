package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHandleProviderWizard_NoStdin_StaticGuide(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	st := chatState{client: nil, providerWizardIn: nil}
	if err := handleProviderWizard(st, &buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "plain REPL") || !strings.Contains(s, "Copy-paste reference") {
		t.Fatalf("expected static guide, got:\n%s", s)
	}
}

func TestRunProviderInteractiveWizard_Cancel(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("\n")
	var buf bytes.Buffer
	if err := runProviderInteractiveWizard(&buf, in, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "cancelled") {
		t.Fatalf("%q", buf.String())
	}
}

func TestRunProviderInteractiveWizard_OpenAI(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("1\n\n\n")
	var buf bytes.Buffer
	if err := runProviderInteractiveWizard(&buf, in, nil); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, `name: openai`) || !strings.Contains(s, "gpt-4o-mini") {
		t.Fatalf("missing openai yaml: %q", s)
	}
}

func TestRunProviderInteractiveWizard_GitHub(t *testing.T) {
	t.Parallel()
	in := strings.NewReader("4\n\n\n")
	var buf bytes.Buffer
	if err := runProviderInteractiveWizard(&buf, in, nil); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, `name: github`) || !strings.Contains(s, "gpt-4o") {
		t.Fatalf("missing github yaml: %q", s)
	}
}
