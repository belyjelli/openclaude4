package providererrs

import "errors"

// ErrCodexNotImplemented indicates the v3 Codex/Copilot protocol is not available on the Go CLI yet.
var ErrCodexNotImplemented = errors.New(`provider "codex" is not implemented in openclaude4 Go.

The v3 TypeScript CLI uses a Codex-specific API that is not available in the Go version.

Alternatives:
  - Use OPENCLAUDE_PROVIDER=openai with a compatible endpoint
  - Use OPENCLAUDE_PROVIDER=ollama for local inference
  - Use OPENCLAUDE_PROVIDER=gemini for Google's Gemini models
  - Run the v3 TypeScript CLI: npm install -g @anthropics/openclaude && openclaude

Contributing: If you need Codex support, consider implementing it as a new provider
following the pattern in internal/providers/openaicomp/. See docs/PROVIDERS.md for guidance.`)
