package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestDescribeEffectiveConfig_NoPanic(t *testing.T) {
	t.Parallel()
	viper.Reset()
	bindViperEnv()
	MergeV3Profile(t.TempDir(), "")
	var buf bytes.Buffer
	DescribeEffectiveConfig(&buf)
	s := buf.String()
	if !strings.Contains(s, "Precedence:") || !strings.Contains(s, "Effective") {
		t.Fatalf("unexpected output:\n%s", s)
	}
}

func TestDescribeEffectiveConfig_WithYAML(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	yml := filepath.Join(tmp, "openclaude.yaml")
	if err := os.WriteFile(yml, []byte("provider:\n  name: ollama\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	viper.Reset()
	Load("")
	var buf bytes.Buffer
	DescribeEffectiveConfig(&buf)
	s := buf.String()
	if !strings.Contains(s, yml) && !strings.Contains(s, "openclaude.yaml") {
		t.Fatalf("expected yaml path in output:\n%s", s)
	}
	if !strings.Contains(s, "ollama") {
		t.Fatalf("expected provider in output:\n%s", s)
	}
}
