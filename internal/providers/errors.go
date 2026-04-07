package providers

import "errors"

// ErrCodexNotImplemented indicates the v3 Codex/Copilot protocol is not available on the Go CLI yet.
var ErrCodexNotImplemented = errors.New(`provider "codex" is not implemented in openclaude4 Go (v3 uses a different Codex API). Use OPENCLAUDE_PROVIDER=openai, gemini, or ollama, or run the v3 TypeScript CLI`)
