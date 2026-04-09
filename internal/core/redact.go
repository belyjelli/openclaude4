package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// RedactedPlaceholder is substituted for detected secret material in logged/kernel event payloads.
const RedactedPlaceholder = "[REDACTED]"

var (
	// Exclude " so curl -H "Authorization: Bearer …" does not absorb the closing quote (then rxAuthHeader would eat to EOL).
	rxBearer = regexp.MustCompile(`(?i)\bBearer\s+[^\s"]+`)
	// Stop before an embedded " so values inside curl -H "Authorization: …" do not swallow the rest of the flag.
	rxAuthHeader = regexp.MustCompile(`(?i)Authorization:\s*[^"\r\n]+`)
	rxEnvAPIKey     = regexp.MustCompile(`(?i)\b(?:OPENAI_API_KEY|GEMINI_API_KEY|GOOGLE_API_KEY|ANTHROPIC_API_KEY|AZURE_OPENAI_API_KEY|OPENROUTER_KEY|OPENROUTER_API_KEY)\s*=\s*\S+`)
	rxOpenAIKey     = regexp.MustCompile(`\bsk-[a-zA-Z0-9]{20,}\b`)
	rxGoogleAPIKey  = regexp.MustCompile(`\bAIza[0-9A-Za-z\-_]{35}\b`)
	rxJSONSecretVal = regexp.MustCompile(`(?i)("(?:api[_-]?key|access[_-]?token|auth[_-]?token|client[_-]?secret|refresh[_-]?token|password|secret|private[_-]?key)"\s*:\s*")([^"\\]*(?:\\.[^"\\]*)*)(")`)
	// Long opaque / base64-like runs (min 80 chars) to catch pasted blobs without nuking short code tokens.
	rxLongOpaque = regexp.MustCompile(`(?:[A-Za-z0-9+/\-_]{4}){20,}={0,2}`)
)

// RedactStringForLog returns s with common credential patterns replaced by RedactedPlaceholder.
// It is best-effort: structured secrets in non-JSON text may slip through; verbose text may false-positive on long alphanumeric runs.
func RedactStringForLog(s string) string {
	if s == "" {
		return s
	}
	out := s
	out = rxBearer.ReplaceAllString(out, "Bearer "+RedactedPlaceholder)
	out = rxAuthHeader.ReplaceAllString(out, "Authorization: "+RedactedPlaceholder)
	out = rxEnvAPIKey.ReplaceAllStringFunc(out, func(m string) string {
		if i := strings.IndexByte(m, '='); i >= 0 {
			return m[:i+1] + RedactedPlaceholder
		}
		return RedactedPlaceholder
	})
	out = rxOpenAIKey.ReplaceAllString(out, RedactedPlaceholder)
	out = rxGoogleAPIKey.ReplaceAllString(out, RedactedPlaceholder)
	out = rxJSONSecretVal.ReplaceAllString(out, `${1}`+RedactedPlaceholder+`${3}`)
	out = rxLongOpaque.ReplaceAllStringFunc(out, func(m string) string {
		if len(m) < 80 {
			return m
		}
		return RedactedPlaceholder
	})
	return out
}

func isSensitiveArgKey(k string) bool {
	kl := strings.ToLower(strings.ReplaceAll(k, "-", "_"))
	switch kl {
	case "api_key", "apikey", "secret", "password", "token", "access_token", "refresh_token",
		"auth_token", "client_secret", "authorization", "bearer", "private_key":
		return true
	default:
		if strings.HasSuffix(kl, "_api_key") || strings.HasSuffix(kl, "_secret") {
			return true
		}
		if strings.HasSuffix(kl, "_token") && kl != "max_tokens" {
			return true
		}
		return false
	}
}

func redactAnyValue(v any) any {
	switch x := v.(type) {
	case string:
		return RedactStringForLog(x)
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, vv := range x {
			if isSensitiveArgKey(k) {
				out[k] = RedactedPlaceholder
			} else {
				out[k] = redactAnyValue(vv)
			}
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, vv := range x {
			out[i] = redactAnyValue(vv)
		}
		return out
	case json.Number:
		return x
	case nil, bool, float64, int, int32, int64, uint, uint32, uint64:
		return x
	default:
		s := fmt.Sprint(v)
		return RedactStringForLog(strings.TrimSpace(strings.ReplaceAll(s, "\n", " ")))
	}
}

// RedactEventForLog returns a copy of e safe to record in transcripts or structured logs (kernel OnEvent path).
// ToolArgs is deep-copied and redacted so the original map used for tool execution is unchanged.
func RedactEventForLog(e Event) Event {
	e.UserText = RedactStringForLog(e.UserText)
	e.TextChunk = RedactStringForLog(e.TextChunk)
	e.AssistantText = RedactStringForLog(e.AssistantText)
	e.Message = RedactStringForLog(e.Message)
	e.ToolArgsJSON = RedactStringForLog(e.ToolArgsJSON)
	e.ToolResultText = RedactStringForLog(e.ToolResultText)
	e.ToolExecError = RedactStringForLog(e.ToolExecError)
	if len(e.ToolArgs) > 0 {
		if cloned, ok := redactAnyValue(e.ToolArgs).(map[string]any); ok {
			e.ToolArgs = cloned
		}
	}
	return e
}

// FormatToolArgsForLog JSON-encodes tool arguments after redaction, for stderr prompts and similar.
func FormatToolArgsForLog(m map[string]any) string {
	if m == nil {
		return "{}"
	}
	red, ok := redactAnyValue(m).(map[string]any)
	if !ok {
		return RedactedPlaceholder
	}
	b, err := json.Marshal(red)
	if err != nil {
		return RedactedPlaceholder
	}
	return string(b)
}
