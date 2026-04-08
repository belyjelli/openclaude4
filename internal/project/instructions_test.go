package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstructionFilenameInDir_none(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if got := InstructionFilenameInDir(dir); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
	if GetProjectInstructionFile(dir) != "" {
		t.Fatal("GetProjectInstructionFile should match empty InstructionFilenameInDir")
	}
	if HasProjectInstructionFile(dir) {
		t.Fatal("HasProjectInstructionFile should be false")
	}
}

func TestInstructionFilenameInDir_claudeOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, FallbackProjectInstructionFile), "# hi\n")
	if got := InstructionFilenameInDir(dir); got != FallbackProjectInstructionFile {
		t.Fatalf("got %q, want %q", got, FallbackProjectInstructionFile)
	}
	if !HasProjectInstructionFile(dir) {
		t.Fatal("expected HasProjectInstructionFile true")
	}
}

func TestInstructionFilenameInDir_agentsOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, PrimaryProjectInstructionFile), "# agents\n")
	if got := InstructionFilenameInDir(dir); got != PrimaryProjectInstructionFile {
		t.Fatalf("got %q, want %q", got, PrimaryProjectInstructionFile)
	}
}

func TestInstructionFilenameInDir_agentsWinsOverClaude(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, PrimaryProjectInstructionFile), "# agents\n")
	mustWriteFile(t, filepath.Join(dir, FallbackProjectInstructionFile), "# claude\n")
	if got := InstructionFilenameInDir(dir); got != PrimaryProjectInstructionFile {
		t.Fatalf("got %q, want %q (AGENTS must win)", got, PrimaryProjectInstructionFile)
	}
}

func TestGetProjectInstructionFilePath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	t.Run("defaults to AGENTS.md when neither exists", func(t *testing.T) {
		t.Parallel()
		want := filepath.Join(dir, PrimaryProjectInstructionFile)
		if got := GetProjectInstructionFilePath(dir); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(sub, FallbackProjectInstructionFile), "# nested\n")
	if got := GetProjectInstructionFilePath(sub); got != filepath.Join(sub, FallbackProjectInstructionFile) {
		t.Fatalf("nested: got %q", got)
	}
}

func TestIsProjectInstructionBasename(t *testing.T) {
	t.Parallel()
	if !IsProjectInstructionBasename(PrimaryProjectInstructionFile) {
		t.Fatal("primary should match")
	}
	if !IsProjectInstructionBasename(FallbackProjectInstructionFile) {
		t.Fatal("fallback should match")
	}
	if IsProjectInstructionBasename("README.md") {
		t.Fatal("unrelated name should not match")
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
