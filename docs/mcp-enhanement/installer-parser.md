**MCP v2 Smart Installer – Expanded Parsing Logic**  
**Part of the MCP v2 Development Plan (no backward compatibility)**  
**Module:** `internal/mcp/installer/parser.go` (and supporting detectors)

This is the **production-grade, extensible parsing engine** that powers the “paste GitHub link → smart install” feature you requested. It turns any public GitHub repo URL into one or more ready-to-use `ServerConfig` candidates with **confidence scores**, auto-detected command, transport, env hints, and approval policy suggestions.

### 1. Design Principles (SOLID + Robustness)
- **Single Responsibility**: Parser only detects → never mutates config or prompts user.
- **Open/Closed**: New detectors added via `Detector` interface (no changes to core pipeline).
- **Strategy Pattern**: Multiple detectors run in parallel; highest-confidence wins (or user chooses).
- **Fail-Safe**: Graceful degradation – if nothing is detected, fall back to “raw Go binary / clone + build” template.
- **Security-First**: Never auto-execute anything; always show exact diff + permission prompt.
- **Performance**: < 2s typical (GitHub raw CDN + in-memory cache; optional GitHub token for higher rate limits).
- **Extensible**: Future support for Smithery registry, Docker Hub, PyPI, etc.

### 2. Input Normalization
```go
type InstallRequest struct {
    URL          string // e.g. https://github.com/modelcontextprotocol/servers or with /tree/main/src/filesystem
    SuggestedName string // optional user override
    DryRun       bool
}
```

Parser normalizes to:
- `owner`, `repo`, `path` (sub-folder), `ref` (branch/tag)
- Validates public repo (HEAD request first)

### 3. Parsing Pipeline (Step-by-Step)

**Step 0: Quick Metadata (parallel, ~200ms)**
- GitHub REST API (`/repos/{owner}/{repo}`) → name, description, topics, language, default branch.
- Topics containing `mcp`, `modelcontextprotocol`, `server` boost score +10.
- Cache key: `repo:owner/repo@ref` (30 min TTL in `~/.openclaude/cache/`).

**Step 1: File Inventory (parallel)**
Fetch only needed raw files via `https://raw.githubusercontent.com/...` (no full clone):
- `README.md` (or `README.markdown`)
- `package.json`
- `go.mod`
- `pyproject.toml` / `setup.py` / `requirements.txt`
- `Cargo.toml` (Rust)
- Any `*.mcp.json`, `mcp.json`, `smithery.json`, `.mcp/manifest.json`
- `Dockerfile` / `docker-compose.yml`
- `server.json` (emerging community standard)

If sub-path specified (e.g. `/tree/main/src/filesystem`), restrict search to that subtree.

**Step 2: Detector Pipeline (parallel, scored)**
Each detector implements:
```go
type Detector interface {
    Name() string
    Detect(ctx context.Context, files map[string][]byte, meta *RepoMetadata) ([]*Candidate, error)
    ConfidenceWeight() float64 // 0.0–1.0
}
```

**Registered Detectors (in priority order)**:

| Detector | Triggers On | Detected Command / Transport | Confidence Boosters | Example Servers |
|----------|-------------|------------------------------|---------------------|-----------------|
| **NPMDetector** (70% of all servers) | `package.json` with name containing `mcp` or `@modelcontextprotocol/server-` or `@brave/` etc. | `npx` or `bunx -y` + package name + args from README code blocks | README has `npx -y @...` or `bunx` | server-filesystem, server-memory, brave-search-mcp-server, github-mcp-server |
| **BunxShortcutDetector** | Same as above + repo uses Bun | Prefer `bunx` over `npx` | — | All official @modelcontextprotocol/* |
| **PythonUVXDetector** | `pyproject.toml` or `mcp-server-` in name | `uvx mcp-server-xxx` or `python -m module` | README mentions `uvx` or `pipx` | mcp-server-git, FastMCP servers |
| **GoBinaryDetector** | `go.mod` + main package | `go run .` or pre-built binary name | — | mcp-k8s-go, custom Go servers |
| **DockerDetector** | `Dockerfile` + `EXPOSE` or `mcp` in labels | `docker run --rm -i ...` | README shows docker example | Some filesystem / enterprise servers |
| **RemoteHTTPDetector** | README contains `https://*.mcp` or `/mcp` endpoint | `transport: http` + URL | GitHub MCP, Copilot remote | github.com/github/github-mcp-server |
| **ManifestDetector** | `*.mcp.json` or `smithery.json` | Direct parse of full config | Highest (100) | Smithery.ai published servers |
| **ReadmeCommandDetector** (fallback) | Regex on README code blocks for `npx`, `bunx`, `uvx`, `docker run`, `go run`, etc. | Extract exact command + args | — | All servers with installation sections |
| **GenericFallback** | Nothing else matches | `git clone && go run .` or `npm install && node build/index.js` | Lowest | Custom / example repos |

**Step 3: Command Extraction from README (smart regex + LLM-light fallback)**
- Look for fenced code blocks (` ```bash`, ```sh
- Regex patterns (ordered by specificity):
  - `npx\s+-y\s+(@[\w-]+/[\w-]+)`
  - `bunx\s+-y\s+...`
  - `uvx\s+mcp-server-...`
  - Full command lines containing `mcp` + server name
- Context-aware: extract following arguments (paths, flags) and map to config `args` + `env` placeholders.

**Step 4: Candidate Ranking & Deduplication**
Each `Candidate`:
```go
type Candidate struct {
    Name          string
    Transport     string // stdio | http | sse | ws
    Command       []string
    Env           map[string]string
    Approval      string // ask | always | never (suggested)
    ExtraArgs     []string
    Confidence    float64 // 0–100
    Reason        string // human-readable e.g. "NPM package detected in package.json + exact npx command in README"
    DetectedFrom  string // detector name
}
```

- Deduplicate by signature (command + args hash)
- Sort by confidence descending
- If multiple > 80, present all to user with radio/select

**Step 5: Post-Processing & Validation**
- Auto-fill `name` from repo name or package name (sanitized: `[^a-z0-9_-]` → `_`)
- Suggest workspace-aware args (e.g. `${workspaceFolder}` for filesystem)
- Env hint detection: `GITHUB_TOKEN`, `BRAVE_API_KEY`, `OPENAI_API_KEY` etc. → mark as `secret: true` in UI prompt
- Security scan: warn if command contains `sudo`, `rm -rf`, etc. (rare but explicit)

### 4. User Permission Flow (after parsing)
1. Show beautiful TUI summary:
   ```
   Found 1 high-confidence configuration for https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem

   Name: filesystem
   Command: npx -y @modelcontextprotocol/server-filesystem ${workspaceFolder}
   Transport: stdio
   Confidence: 98
   Reason: Official NPM package + exact command in README
   ```
2. Diff of what will be written to config layer (project > user > enterprise)
3. Prompt:
   - Approval policy: [ask] / always / never
   - Add to: Project config / User config
   - Any env vars to set now? (masked input)
   - [Install & Connect] / [Edit manually] / [Cancel]
4. On confirm → write to correct config layer → immediately `Manager.Connect()` (with dry-run flag if requested)

### 5. Edge Cases & Robustness
- Private repos → optional `--github-token` flag (stored in keyring).
- Monorepo / sub-folder → respect `/tree/...` path.
- No README → still works via `package.json` / `go.mod`.
- Rate-limited → fallback to cached data + clear error.
- Malformed repo → friendly “Not a valid MCP server repo” + link to awesome-mcp list.
- Future: Smithery.ai registry lookup (one extra HTTP call if GitHub parse fails).

### 6. Implementation Notes (Go)
- Use `github.com/google/go-github/v62` (optional, rate-limit friendly) + raw `net/http` for files.
- `golang.org/x/net/html` + `regexp` for README parsing.
- All detectors in `installer/detectors/` sub-package.
- Tests: golden-file tests against real repos (filesystem, brave-search, example-remote-server, custom Go servers).

This parsing logic makes the installer **feel magical** while staying 100% transparent and secure — exactly what you asked for.

It slots perfectly into **Phase 4** of the MCP v2 roadmap I gave earlier.

**Next step?**  
I can now write the full `parser.go` + first detectors (NPM + ReadmeCommand) as a ready-to-review PR skeleton, or expand any detector further.

Just say the word and we start coding MCP v2 installer today.