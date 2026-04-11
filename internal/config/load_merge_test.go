package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestImplicitMerge_HomeOpenClaudeOnly(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(cwd)

	xdg := filepath.Join(home, ".config", "openclaude")
	if err := os.MkdirAll(xdg, 0o755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(xdg, "openclaude.yaml")
	if err := os.WriteFile(base, []byte("provider:\n  name: openai\n  model: from-home-base\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	Load("")

	if got := Model(); got != "from-home-base" {
		t.Fatalf("Model() = %q, want from-home-base", got)
	}
	if MergedHomeOpenClaudePath != base {
		t.Fatalf("MergedHomeOpenClaudePath = %q, want %q", MergedHomeOpenClaudePath, base)
	}
	if MergedHomeOpenClaudeV4Path != "" {
		t.Fatalf("MergedHomeOpenClaudeV4Path should be empty, got %q", MergedHomeOpenClaudeV4Path)
	}
	if MergedCwdOpenClaudePath != "" {
		t.Fatalf("MergedCwdOpenClaudePath should be empty, got %q", MergedCwdOpenClaudePath)
	}
}

func TestImplicitMerge_OpenClaudeV4OverridesHomeOpenClaude(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(cwd)

	xdg := filepath.Join(home, ".config", "openclaude")
	if err := os.MkdirAll(xdg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xdg, "openclaude.yaml"), []byte("provider:\n  name: openai\n  model: model-A\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	v4path := filepath.Join(xdg, "openclaudev4.yaml")
	if err := os.WriteFile(v4path, []byte("provider:\n  name: openai\n  model: model-B\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	Load("")

	if got := Model(); got != "model-B" {
		t.Fatalf("Model() = %q, want model-B (openclaudev4 should override)", got)
	}
	if MergedHomeOpenClaudeV4Path != v4path {
		t.Fatalf("MergedHomeOpenClaudeV4Path = %q, want %q", MergedHomeOpenClaudeV4Path, v4path)
	}
}

func TestImplicitMerge_CwdOverridesHomeStack(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(cwd)

	xdg := filepath.Join(home, ".config", "openclaude")
	if err := os.MkdirAll(xdg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xdg, "openclaude.yaml"), []byte("provider:\n  name: openai\n  model: model-A\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xdg, "openclaudev4.yaml"), []byte("provider:\n  name: openai\n  model: model-B\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cwdYAML := filepath.Join(cwd, "openclaude.yaml")
	if err := os.WriteFile(cwdYAML, []byte("provider:\n  name: openai\n  model: model-C\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	Load("")

	if got := Model(); got != "model-C" {
		t.Fatalf("Model() = %q, want model-C (cwd should win)", got)
	}
	if MergedCwdOpenClaudePath != cwdYAML {
		t.Fatalf("MergedCwdOpenClaudePath = %q, want %q", MergedCwdOpenClaudePath, cwdYAML)
	}
}

func TestWritableConfigPath_ImplicitOrder(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(cwd)
	viper.Reset()
	Load("")

	xdg := filepath.Join(home, ".config", "openclaude")
	if err := os.MkdirAll(xdg, 0o755); err != nil {
		t.Fatal(err)
	}

	// No files yet → default under XDG
	wantDefault := filepath.Join(xdg, "openclaude.yaml")
	p, err := WritableConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(p) != filepath.Clean(wantDefault) {
		t.Fatalf("WritableConfigPath = %q, want %q", p, wantDefault)
	}

	onlyBase := filepath.Join(xdg, "openclaude.yaml")
	if err := os.WriteFile(onlyBase, []byte("provider:\n  name: openai\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err = WritableConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(p) != filepath.Clean(onlyBase) {
		t.Fatalf("with only home openclaude: got %q, want %q", p, onlyBase)
	}

	onlyV4 := filepath.Join(xdg, "openclaudev4.yml")
	if err := os.WriteFile(onlyV4, []byte("provider:\n  name: openai\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(onlyBase)
	p, err = WritableConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(p) != filepath.Clean(onlyV4) {
		t.Fatalf("with only home openclaudev4: got %q, want %q", p, onlyV4)
	}

	// Both home files: v4 wins when no cwd file
	if err := os.WriteFile(onlyBase, []byte("provider:\n  name: openai\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err = WritableConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(p) != filepath.Clean(onlyV4) {
		t.Fatalf("with both home files: got %q, want openclaudev4 %q", p, onlyV4)
	}

	cwdYAML := filepath.Join(cwd, "openclaude.yaml")
	if err := os.WriteFile(cwdYAML, []byte("provider:\n  name: openai\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err = WritableConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(p) != filepath.Clean(cwdYAML) {
		t.Fatalf("with cwd yaml: got %q, want %q", p, cwdYAML)
	}
}

func TestWritableConfigPath_ExplicitConfig(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	explicit := filepath.Join(cwd, "custom.yaml")
	if err := os.WriteFile(explicit, []byte("provider:\n  name: openai\n  model: from-explicit\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	t.Chdir(cwd)

	cwdLocal := filepath.Join(cwd, "openclaude.yaml")
	if err := os.WriteFile(cwdLocal, []byte("provider:\n  name: openai\n  model: from-cwd\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	Load(explicit)

	p, err := WritableConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	expAbs, _ := filepath.Abs(explicit)
	if filepath.Clean(p) != filepath.Clean(expAbs) {
		t.Fatalf("WritableConfigPath with --config file: got %q, want %q", p, expAbs)
	}
}
