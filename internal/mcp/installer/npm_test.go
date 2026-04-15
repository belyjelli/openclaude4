package installer

import (
	"context"
	"testing"
)

func TestNPMDetector_basic(t *testing.T) {
	meta := &RepoMetadata{Owner: "x", Repo: "y", Ref: "main"}
	files := map[string][]byte{
		"package.json": []byte(`{"name":"@modelcontextprotocol/server-memory","description":"MCP","keywords":["mcp"]}`),
		"README.md":    []byte("# x\n\nRun:\n\n```bash\nnpx -y @modelcontextprotocol/server-memory\n```\n"),
	}
	var d NPM
	got, err := d.Detect(context.Background(), files, meta)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 1 {
		t.Fatalf("expected candidates, got %d", len(got))
	}
	if got[0].Name == "" || len(got[0].Command) < 2 {
		t.Fatalf("bad candidate: %#v", got[0])
	}
}
