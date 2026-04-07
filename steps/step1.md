**Review of https://github.com/belyjelli/openclaude4**

This repo is **not yet a working project** — it is explicitly a **planning / design-stage repository** for **OpenClaude v4**, positioned as the “Golang binary” rewrite of the original OpenClaude CLI.

**Current state (as of April 2026):**
- Zero published code, no binaries, no releases, no stars/forks.
- It contains only documentation files (README, DESIGN.md, ROADMAP, etc.).
- The goal stated in the title and docs is to create a **native single-binary Golang implementation** of OpenClaude (the popular 19k-star TS/Bun coding agent) for better performance, easier distribution, smaller footprint, and full cross-platform support without needing Node/Bun.

The repo’s real value right now is its **high-quality architecture documentation**. It outlines a clean, modular design for v4 that is significantly better than the current TypeScript monolith. It emphasizes:
- Strict separation of concerns (core engine, providers, tools, agents, UI layer).
- Extensible provider system.
- Tool/MCP execution sandboxing.
- TUI + headless gRPC modes.
- Roadmap for incremental feature parity with the current v3 TS version.

**Verdict**: Excellent blueprint and motivation, but no code yet. This is the perfect reference document set if you want to build the “real” full-fledged Golang version.

---

**The source you want to port: https://github.com/Gitlawb/openclaude**

This is the **mature, production-grade** original (18.9k stars). It is a **TypeScript + Bun** terminal-first coding agent that works with OpenAI-compatible APIs, Gemini, DeepSeek, Ollama, GitHub Models, etc. It supports:
- Rich slash commands (`/provider`, `/onboard-github`, etc.)
- Full tool ecosystem (bash, file read/write/edit, grep, glob, web search/fetch, MCP, agents, tasks)
- Streaming + multi-turn tool-calling loop
- Provider profiles + settings.json routing
- VS Code extension
- Headless gRPC server mode

It is **not** written in Go — that’s why you (and belyjelli) want a native Go rewrite.

---

**How to build a complete, full-fledged Golang repo (OpenClaude-Go / openclaude4)**

Here is a **practical, production-ready roadmap** that follows the design principles from belyjelli/openclaude4 + the feature set of Gitlawb/openclaude.

### 1. Project Setup (Day 1)

```bash
mkdir openclaude-go && cd openclaude-go
go mod init github.com/yourname/openclaude-go
go get -u github.com/spf13/cobra@latest
go get -u github.com/spf13/viper@latest
go get -u github.com/charmbracelet/bubbletea@latest
go get -u github.com/charmbracelet/lipgloss@latest
go get -u github.com/charmbracelet/bubbles@latest
go get -u github.com/sashabaranov/go-openai@latest
go get -u google.golang.org/grpc@latest
# Add more as needed: ollama-go, google/generative-ai-go, etc.
```

**Recommended folder structure** (adapted from belyjelli DESIGN.md):

```
openclaude-go/
├── cmd/
│   └── openclaude/          # main entrypoint + cobra root
├── internal/
│   ├── core/                # engine, agent loop, tool dispatcher
│   ├── providers/           # OpenAI, Gemini, Ollama, GitHub Models, etc.
│   ├── tools/               # bash, fs, grep, web, mcp, etc.
│   ├── tui/                 # Bubble Tea model + views
│   ├── config/              # viper + profiles
│   ├── grpc/                # headless server (proto + server)
│   └── utils/               # streaming, sandbox, etc.
├── pkg/                     # public packages if you want to expose library
├── proto/                   # openclaude.proto (copy from original)
├── .github/workflows/       # CI + Goreleaser
├── README.md
├── DESIGN.md                # copy/adapt from belyjelli/openclaude4
├── ROADMAP.md
├── go.mod
├── goreleaser.yml
└── settings.example.json
```

### 2. Core Stack Choices (best-in-class Go equivalents)

| Original (TS)       | Go Replacement                     | Why |
|---------------------|------------------------------------|-----|
| Commander / Ink     | Cobra + Bubbletea + Lipgloss       | Mature, beautiful TUI |
| Config / profiles   | Viper + `~/.openclaude-profile.json` | Same UX |
| OpenAI-compatible   | `sashabaranov/go-openai`           | Best client, custom BaseURL |
| Ollama              | `ollama/ollama` Go client or HTTP | Official |
| Gemini              | `google/generative-ai-go`          | Official |
| Streaming + tools   | Custom loop with channels          | Same multi-turn logic |
| gRPC headless       | `google.golang.org/grpc`           | Drop-in |
| Builds              | Goreleaser                         | Single binary releases |

### 3. Phased Implementation Plan (follow belyjelli ROADMAP)

**Phase 0 – MVP (1 weekend)**
- Cobra CLI skeleton with `/provider` and basic chat
- One provider (OpenAI-compatible via go-openai)
- Simple prompt → streaming response (no tools yet)

**Phase 1 – Tools & Agent Loop**
- Implement core tools: `read`, `write`, `bash`, `grep`, `glob`, `web_search` (DuckDuckGo or Firecrawl)
- Tool-calling loop (exactly like original)
- Sandboxing (use `os/exec` with strict limits or `secure-exec` style)

**Phase 2 – Full Providers & Profiles**
- Gemini, Ollama, GitHub Models, DeepSeek, etc.
- Profile system + `~/.openclaude-profile.json`
- Agent routing from `settings.json`

**Phase 3 – Polish & Extras**
- Rich TUI with Bubble Tea (same look & feel as original Ink UI)
- MCP support (Model Context Protocol)
- Headless gRPC server
- VS Code extension (separate repo or sub-module, can stay TS or use Go LSP if you want)

**Phase 4 – Production**
- Comprehensive tests
- Goreleaser + GitHub Releases (Linux, macOS, Windows, ARM)
- Homebrew formula, Scoop, AUR, etc.
- Documentation + examples

### 4. Key Technical Gotchas & Solutions

- **Tool execution security** → Never run raw bash without user confirmation + timeout + working-directory restriction (copy the safety patterns from the original Python helpers).
- **Streaming** → Use `go-openai`’s `ChatCompletionStream` + channels to feed Bubble Tea.
- **Cross-provider differences** → Abstract `Provider` interface with `Chat` and `ToolCall` methods; handle output limits per model.
- **Single binary** → Goreleaser + `CGO_ENABLED=0` → tiny static binaries.

**Goreleaser example** (basic `goreleaser.yml`):
```yaml
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
```

### 5. Recommended Next Actions (copy-paste)

1. Create the repo: `github.com/yourname/openclaude-go`
2. Copy the DESIGN.md + ROADMAP.md from belyjelli/openclaude4 into yours.
3. Run the setup commands above.
4. Start with `cmd/openclaude/main.go` + Cobra root + one provider.
5. When you have the basic chat working, ping me and I can give you the exact tool-calling loop code + Bubble Tea skeleton.

---

## Implementation status (this repository)

The following items from **§1 Project Setup** and **Phase 0 – MVP** are implemented here:

| Item | Status |
|------|--------|
| `go.mod` + Cobra + Viper + `go-openai` | Done (`github.com/gitlawb/openclaude4`) |
| Layout: `cmd/openclaude`, `internal/config`, `internal/providers`, stubs under `internal/core`, `tools`, `tui`, `grpc`, `utils` | Done |
| `goreleaser.yml`, `settings.example.json`, `.github/workflows/go.yml` | Done |
| CLI: default command = streaming chat REPL | Done |
| `/provider` (plus `/help`, `/clear`, `/exit`) | Done |
| `openclaude version` | Done |
| Bubble Tea / Lipgloss TUI | Not started (Phase 3 in this doc) |
| gRPC / `proto/` | Not started |

**Run:** set `OPENAI_API_KEY`, then `go run ./cmd/openclaude` or build per [README.md](../README.md).

**Next:** Phase 1–2 Go slices live in this repo (see [steps/step2.md](./step2.md), [docs/CONFIG.md](../docs/CONFIG.md)). Roadmap **Phase 3**: MCP, slash commands, deeper tool parity ([docs/ROADMAP.md](../docs/ROADMAP.md)).