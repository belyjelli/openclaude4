**Phase 1 – Tools & Agent Loop** (detailed implementation guide for openclaude-go)

This phase gets you a **fully functional coding agent** that can:
- Read/write/edit files safely
- Run shell commands (`bash` / PowerShell on Windows)
- Search code with `grep` and `glob`
- Perform web searches
- Run the full **multi-turn tool-calling loop** (exactly like the original OpenClaude)

We follow the clean architecture from belyjelli/openclaude4’s DESIGN.md and mirror the original `src/tools/` structure.

### 1. Directory Structure (add to your repo)

```
internal/
├── tools/
│   ├── tool.go                 # Tool interface + registry
│   ├── file_read.go
│   ├── file_write.go
│   ├── file_edit.go
│   ├── bash.go
│   ├── grep.go
│   ├── glob.go
│   ├── web_search.go
│   └── registry.go             # All tools registered here
├── core/
│   └── agent.go                # Tool-calling loop + engine
└── sandbox/                    # (optional but recommended) execution sandbox
```

### 2. Core Tool Interface (`internal/tools/tool.go`)

```go
package tools

import (
	"context"
	"fmt"
)

// Tool defines the contract every tool must implement
type Tool interface {
	// Name is the exact string the model will call (e.g. "Bash", "FileRead")
	Name() string
	// Description is shown to the model so it knows when to use the tool
	Description() string
	// Parameters is a JSON Schema (map) describing the arguments
	Parameters() map[string]any
	// Execute runs the tool. Returns result string + optional error
	Execute(ctx context.Context, args map[string]any) (string, error)
	// IsDangerous returns true for tools that need user confirmation (bash, write, edit)
	IsDangerous() bool
}

// Registry holds all available tools
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	r := &Registry{tools: make(map[string]Tool)}
	// Register all Phase 1 tools here (see registry.go)
	return r
}

func (r *Registry) Register(t Tool) { r.tools[t.Name()] = t }
func (r *Registry) Get(name string) (Tool, bool) { t, ok := r.tools[name]; return t, ok }
func (r *Registry) List() []Tool {
	var list []Tool
	for _, t := range r.tools { list = append(list, t) }
	return list
}
```

### 3. Individual Tool Implementations

#### FileReadTool (`internal/tools/file_read.go`)
```go
type FileRead struct{}

func (FileRead) Name() string       { return "FileRead" }
func (FileRead) Description() string { return "Reads the complete content of a file. Use for inspecting code." }
func (FileRead) IsDangerous() bool  { return false }

func (FileRead) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{"type": "string", "description": "Absolute or relative path to the file"},
		},
		"required": []string{"file_path"},
	}
}

func (FileRead) Execute(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["file_path"].(string)
	if path == "" { return "", fmt.Errorf("file_path is required") }
	
	content, err := os.ReadFile(path)
	if err != nil { return "", err }
	return string(content), nil
}
```

#### FileWriteTool & FileEditTool (`internal/tools/file_write.go` + `file_edit.go`)
**FileWrite** = overwrite/create entire file  
**FileEdit** = smart patch (recommended for large files – use diff-style or line-range)

```go
// FileWriteTool (simple overwrite)
type FileWrite struct{}

func (FileWrite) Name() string { return "FileWrite" }
func (FileWrite) IsDangerous() bool { return true } // always confirm

// ... Parameters: file_path + content

func (FileWrite) Execute(_ context.Context, args map[string]any) (string, error) {
	path := args["file_path"].(string)
	content := args["content"].(string)
	
	// Safety: never allow paths outside project root (add check)
	if err := validatePathWithinProject(path); err != nil {
		return "", err
	}
	
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("✓ Wrote %d bytes to %s", len(content), path), nil
}
```

**FileEditTool** – use a simple diff or line-based edit (original uses a patch format). For MVP you can start with `file_path + old_string + new_string` replacement, then upgrade to proper unified diff later.

#### BashTool (`internal/tools/bash.go`)
```go
type Bash struct{}

func (Bash) Name() string       { return "Bash" }
func (Bash) IsDangerous() bool  { return true }

func (Bash) Parameters() map[string]any { /* command + optional cwd, timeout */ }

func (Bash) Execute(ctx context.Context, args map[string]any) (string, error) {
	cmdStr := args["command"].(string)
	
	// Sandboxing (highly recommended)
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Use secure exec helper (see sandbox/ below)
	output, err := sandbox.RunCommand(execCtx, cmdStr, getWorkingDir())
	return output, err
}
```

**PowerShellTool** on Windows is almost identical (use `powershell.exe`).

#### GrepTool (`internal/tools/grep.go`)
Uses Go’s `regexp` or calls `rg` (ripgrep) if installed (fastest).

```go
type Grep struct{}

func (Grep) Name() string { return "Grep" }
// Parameters: pattern, path (default "."), include (glob)

func (Grep) Execute(_ context.Context, args map[string]any) (string, error) {
	pattern := args["pattern"].(string)
	path := getStringOrDefault(args, "path", ".")
	
	// Simple version: use filepath.Walk + regexp
	// Better version: exec.Command("rg", "--json", pattern, path) if rg is present
	// Fall back to pure Go implementation
	return runGrepPureGo(pattern, path)
}
```

#### GlobTool (`internal/tools/glob.go`)
```go
type Glob struct{}

func (Glob) Name() string { return "Glob" }
// Parameters: pattern e.g. "**/*.go"

func (Glob) Execute(_ context.Context, args map[string]any) (string, error) {
	pattern := args["pattern"].(string)
	matches, err := filepath.Glob(pattern)
	// or use doublestar for ** support
	return strings.Join(matches, "\n"), err
}
```

#### WebSearchTool (`internal/tools/web_search.go`)
Simple & privacy-friendly implementation:

```go
type WebSearch struct{}

func (WebSearch) Name() string { return "WebSearch" }

func (WebSearch) Execute(ctx context.Context, args map[string]any) (string, error) {
	query := args["query"].(string)
	// Option 1: DuckDuckGo HTML scraper (no API key)
	// Option 2: Use free tier of Serper.dev / Tavily / Firecrawl
	results, err := duckduckgo.Search(ctx, query, 5)
	return formatSearchResults(results), err
}
```

(You can add `WebFetchTool` similarly using `net/http` + `github.com/PuerkitoBio/goquery` for markdown conversion.)

### 4. Registry (`internal/tools/registry.go`)

```go
func NewDefaultRegistry() *tools.Registry {
	r := tools.NewRegistry()
	r.Register(tools.FileRead{})
	r.Register(tools.FileWrite{})
	r.Register(tools.FileEdit{})
	r.Register(tools.Bash{})
	r.Register(tools.Grep{})
	r.Register(tools.Glob{})
	r.Register(tools.WebSearch{})
	// Add more later
	return r
}
```

### 5. Sandboxing (Critical – `internal/sandbox/sandbox.go`)

```go
package sandbox

// RunCommand runs shell commands with strict limits
func RunCommand(ctx context.Context, command string, cwd string) (string, error) {
	// 1. Parse command to block dangerous things (rm -rf /, etc.)
	if isDangerousCommand(command) {
		return "", fmt.Errorf("command blocked for safety")
	}
	
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = cwd
	cmd.Env = safeEnvironment()
	
	output, err := cmd.CombinedOutput()
	return string(output), err
}
```

Add a simple `isDangerousCommand` checker (blacklist common destructive patterns).

### 6. The Agent/Tool-Calling Loop (`internal/core/agent.go`)

This is the heart of Phase 1:

```go
func (a *Agent) Run(ctx context.Context, userPrompt string) error {
	messages := []openai.ChatCompletionMessage{{Role: "user", Content: userPrompt}}

	for {
		// 1. Call model with current tools
		resp, err := a.provider.ChatWithTools(ctx, messages, a.registry.List())
		if err != nil { return err }

		// 2. Add assistant message
		messages = append(messages, resp.Message)

		// 3. If no tool calls → done
		if len(resp.ToolCalls) == 0 {
			// Stream final answer to TUI
			a.tui.Stream(resp.Content)
			return nil
		}

		// 4. Execute each tool call (with user confirmation for dangerous ones)
		for _, tc := range resp.ToolCalls {
			tool, ok := a.registry.Get(tc.Name)
			if !ok { continue }

			if tool.IsDangerous() {
				if !a.tui.ConfirmToolExecution(tc.Name, tc.Args) {
					continue // user refused
				}
			}

			result, err := tool.Execute(ctx, tc.Args)
			resultMsg := fmt.Sprintf("Tool %s returned:\n%s", tc.Name, result)
			if err != nil { resultMsg += "\nError: " + err.Error() }

			// 5. Feed result back to model
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    "tool",
				Content: resultMsg,
				ToolCallID: tc.ID,
			})
		}
	}
}
```

### Next Steps After This Phase

Once you have the above:
- Hook it into your Bubble Tea TUI (streaming + confirmation dialogs)
- Extend `internal/providers/provider.go` with a first-class `ChatWithTools` / stream interface shared by multiple providers
- Test the full loop with a simple prompt like “create a hello.go file and run it”

---

## Implementation status (this repository)

| Item | Status |
|------|--------|
| `internal/tools` — interface, registry, OpenAI schema export | Done |
| `FileRead`, `FileWrite`, `FileEdit`, `Bash`, `Grep`, `Glob`, `WebSearch` | Done |
| `internal/sandbox` — shell runner + basic blocklist | Done |
| `internal/core` — streaming accumulation + multi-turn tool loop | Done |
| `openaicomp.StreamChatWithTools` | Done |
| `cmd/openclaude` REPL wired to agent + workdir context | Done |
| Bubble Tea TUI | Not started |
| Dedicated `internal/sandbox/` syscall-level isolation | Not started (shell limits only) |

Run: `go run ./cmd/openclaude` from the repo (set `OPENAI_API_KEY`). See [README.md](../README.md) for env vars.

---

## Phase 2 additions (this repository)

| Item | Status |
|------|--------|
| Config file search (`openclaude.yaml` / json, `--config`) | Done — [docs/CONFIG.md](../docs/CONFIG.md) |
| Env + flag merge via Viper | Done |
| Second provider: **Ollama** (OpenAI-compatible `/v1` on host) | Done |
| `openclaude doctor` | Done |
| Gemini / Codex providers | Not started |
| Import `.openclaude-profile.json` automatically | Not started |
| CI tests with `httptest` mock API | Partial — `internal/core/agent_test.go` + `internal/providers/ping_test.go` |