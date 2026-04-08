// Package startupbanner renders the colorful CLI splash (OpenClaude v3–style ANSI art).
package startupbanner

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/providers"
	"github.com/mattn/go-isatty"
)

const esc = "\x1b["

type rgb [3]int

func (c rgb) ansiFG() string {
	return fmt.Sprintf("%s38;2;%d;%d;%dm", esc, c[0], c[1], c[2])
}

const reset = esc + "0m"
const dim = esc + "2m"

func lerp(a, b rgb, t float64) rgb {
	t = math.Max(0, math.Min(1, t))
	return rgb{
		int(math.Round(float64(a[0]) + (float64(b[0])-float64(a[0]))*t)),
		int(math.Round(float64(a[1]) + (float64(b[1])-float64(a[1]))*t)),
		int(math.Round(float64(a[2]) + (float64(b[2])-float64(a[2]))*t)),
	}
}

func gradAt(stops []rgb, t float64) rgb {
	t = math.Max(0, math.Min(1, t))
	s := t * float64(len(stops)-1)
	i := int(math.Floor(s))
	if i >= len(stops)-1 {
		return stops[len(stops)-1]
	}
	return lerp(stops[i], stops[i+1], s-float64(i))
}

func paintLine(text string, stops []rgb, lineT float64) string {
	if len(text) == 0 {
		return ""
	}
	var b strings.Builder
	for i, r := range text {
		var t float64
		if len(text) > 1 {
			t = lineT*0.5 + (float64(i)/float64(len(text)-1))*0.5
		}
		c := gradAt(stops, t)
		b.WriteString(c.ansiFG())
		b.WriteRune(r)
	}
	b.WriteString(reset)
	return b.String()
}

var sunsetGrad = []rgb{
	{255, 180, 100},
	{240, 140, 80},
	{217, 119, 87},
	{193, 95, 60},
	{160, 75, 55},
	{130, 60, 50},
}

var (
	accent  = rgb{240, 148, 100}
	cream   = rgb{220, 195, 170}
	dimCol  = rgb{120, 100, 82}
	borderC = rgb{100, 80, 65}
	localHi = rgb{130, 175, 130}
)

var logoOpen = []string{
	"  ████████╗ ████████╗ ███████╗ ██╗  ██╗",
	"  ██╔═══██║ ██╔═══██║ ██╔════╝ ███╗ ██║",
	"  ██║   ██║ ████████║ ██████╗  ████╗██║",
	"  ██║   ██║ ██╔═════╝ ██╔═══╝  ██╔████║",
	"  ████████║ ██║       ███████╗ ██║╚███║",
	"  ╚═══════╝ ╚═╝       ╚══════╝ ╚═╝ ╚══╝",
}

var logoClaude = []string{
	"  ████████╗ ██╗      ████████╗ ██╗   ██╗ ████████╗ ████████╗",
	"  ██╔═════╝ ██║      ██╔═══██║ ██║   ██║ ██╔═══██║ ██╔════╝",
	"  ██║       ██║      ████████║ ██║   ██║ ██║   ██║ ██████╗  ",
	"  ██║       ██║      ██╔═══██║ ██║   ██║ ██║   ██║ ██╔═══╝  ",
	"  ████████╗ ███████╗ ██║   ██║ ╚██████╔╝ ████████║ ████████╗",
	"  ╚═══════╝ ╚══════╝ ╚═╝   ╚═╝  ╚═════╝  ╚══════╝ ╚══════╝",
}

func boxRow(content string, width, rawLen int) string {
	pad := width - 2 - rawLen
	if pad < 0 {
		pad = 0
	}
	return borderC.ansiFG() + "│" + reset + content + strings.Repeat(" ", pad) + borderC.ansiFG() + "│" + reset
}

func providerLabel(kind, baseURL, model string) (name string, isLocal bool) {
	b := strings.ToLower(baseURL)
	m := strings.ToLower(model)
	isLocal = kind == "ollama" ||
		strings.Contains(b, "localhost") ||
		strings.Contains(b, "127.0.0.1") ||
		strings.Contains(b, "0.0.0.0")

	switch kind {
	case "ollama":
		return "Ollama", true
	case "gemini":
		return "Google Gemini", false
	case "github":
		return "GitHub Models", false
	case "openai":
		switch {
		case strings.Contains(b, "deepseek") || strings.Contains(m, "deepseek"):
			return "DeepSeek", isLocal
		case strings.Contains(b, "openrouter"):
			return "OpenRouter", false
		case strings.Contains(b, "api.x.ai") || strings.Contains(b, "x.ai"):
			return "xAI", false
		case strings.Contains(b, "fireworks.ai") || strings.Contains(b, "fireworks"):
			return "Fireworks AI", false
		case strings.Contains(b, "cerebras"):
			return "Cerebras", false
		case strings.Contains(b, "perplexity"):
			return "Perplexity", false
		case strings.Contains(b, "together"):
			return "Together AI", false
		case strings.Contains(b, "groq"):
			return "Groq", false
		case strings.Contains(b, "mistral"):
			return "Mistral", false
		case strings.Contains(b, "azure") || strings.Contains(b, "openai.azure.com"):
			return "Azure OpenAI", false
		case strings.Contains(b, "bedrock"):
			return "AWS Bedrock", false
		case strings.Contains(b, "cohere"):
			return "Cohere", false
		case strings.Contains(b, "replicate"):
			return "Replicate", false
		case strings.Contains(b, "anyscale"):
			return "Anyscale", false
		case strings.Contains(b, "nebius"):
			return "Nebius", false
		case strings.Contains(b, "siliconflow"):
			return "SiliconFlow", false
		case strings.Contains(b, "hyperbolic"):
			return "Hyperbolic", false
		case strings.Contains(b, "lepton"):
			return "Lepton", false
		case strings.Contains(b, "nvidia"):
			return "NVIDIA NIM", false
		default:
			if strings.Contains(m, "llama") {
				return "Meta Llama", isLocal
			}
			if isLocal {
				return "Local (OpenAI-compatible)", true
			}
			return "OpenAI", false
		}
	default:
		if kind != "" {
			return strings.ToUpper(kind[:1]) + kind[1:], isLocal
		}
		return "Provider", isLocal
	}
}

// Render builds the full ANSI splash (trailing newline not included).
func Render(client core.StreamClient, version, mcpLine string) string {
	info, ok := providers.AsStreamClientInfo(client)
	if !ok {
		return plainFallback(client, version, mcpLine)
	}

	pName, isLocal := providerLabel(info.ProviderKind(), info.BaseURL(), info.Model())
	model := info.Model()
	baseURL := info.BaseURL()
	if baseURL == "" {
		baseURL = "(default)"
	}
	const maxEp = 38
	ep := baseURL
	if len(ep) > maxEp {
		ep = ep[:35] + "..."
	}

	const W = 62
	var out []string
	out = append(out, "")

	var logoLines []string
	logoLines = append(logoLines, logoOpen...)
	logoLines = append(logoLines, "")
	logoLines = append(logoLines, logoClaude...)
	total := len(logoLines)
	for i, line := range logoLines {
		if line == "" {
			out = append(out, "")
			continue
		}
		var t float64
		if total > 1 {
			t = float64(i) / float64(total-1)
		}
		out = append(out, paintLine(line, sunsetGrad, t))
	}

	out = append(out, "")
	out = append(out, "  "+accent.ansiFG()+"✦"+reset+" "+cream.ansiFG()+"Any model. Every tool. Zero limits."+reset+" "+accent.ansiFG()+"✦"+reset)
	out = append(out, "")

	out = append(out, borderC.ansiFG()+"╔"+strings.Repeat("═", W-2)+"╗"+reset)

	lbl := func(k, v string, vc rgb) (string, int) {
		padK := fmt.Sprintf("%-9s", k)
		s := " " + dim + dimCol.ansiFG() + padK + reset + " " + vc.ansiFG() + v + reset
		raw := " " + padK + " " + v
		return s, len([]rune(raw))
	}

	provC := accent
	if isLocal {
		provC = localHi
	}
	r, l := lbl("Provider", pName, provC)
	out = append(out, boxRow(r, W, l))
	r, l = lbl("Model", model, cream)
	out = append(out, boxRow(r, W, l))
	r, l = lbl("Endpoint", ep, cream)
	out = append(out, boxRow(r, W, l))

	out = append(out, borderC.ansiFG()+"╠"+strings.Repeat("═", W-2)+"╣"+reset)

	sC := accent
	sL := "cloud"
	if isLocal {
		sC = localHi
		sL = "local"
	}
	// — is U+2014
	sRow := " " + sC.ansiFG() + "●" + reset + " " + dim + dimCol.ansiFG() + sL + reset + "    " + dim + dimCol.ansiFG() + "Ready — type " + reset + accent.ansiFG() + "/help" + reset + dim + dimCol.ansiFG() + " to begin" + reset
	sLen := len([]rune(" ● " + sL + "    Ready — type /help to begin"))
	out = append(out, boxRow(sRow, W, sLen))

	out = append(out, borderC.ansiFG()+"╚"+strings.Repeat("═", W-2)+"╝"+reset)
	out = append(out, "  "+dim+dimCol.ansiFG()+"openclaude "+reset+accent.ansiFG()+"v"+version+reset)

	if strings.TrimSpace(mcpLine) != "" {
		out = append(out, "")
		out = append(out, "  "+dim+dimCol.ansiFG()+mcpLine+reset)
	}
	out = append(out, "")

	return strings.Join(out, "\n")
}

func plainFallback(client core.StreamClient, version, mcpLine string) string {
	var b strings.Builder
	if info, ok := providers.AsStreamClientInfo(client); ok {
		pretty, _ := providerLabel(info.ProviderKind(), info.BaseURL(), info.Model())
		_, _ = fmt.Fprintf(&b, "OpenClaude %s (phase 3). Provider: %s. Model: %s. Type /help. Ctrl+D to exit.\n",
			version, pretty, info.Model())
	} else {
		_, _ = fmt.Fprintf(&b, "OpenClaude %s (phase 3). Type /help. Ctrl+D to exit.\n", version)
	}
	if strings.TrimSpace(mcpLine) != "" {
		_, _ = fmt.Fprintln(&b, mcpLine)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func writerIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok || f == nil {
		return false
	}
	return isatty.IsTerminal(uintptr(f.Fd()))
}

// SplashDisabled is true when OPENCLAUDE_NO_SPLASH requests a plain one-line banner (no ANSI art).
func SplashDisabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("OPENCLAUDE_NO_SPLASH")))
	return v == "1" || v == "true" || v == "yes"
}

// UseANSISplashFor reports whether we should render truecolor ANSI art for output written to w
// (stderr TTY, not CI, not OPENCLAUDE_NO_SPLASH).
func UseANSISplashFor(w io.Writer) bool {
	if os.Getenv("CI") != "" {
		return false
	}
	if !writerIsTTY(w) || SplashDisabled() {
		return false
	}
	return true
}

// BannerContent returns either the full ANSI splash or the plain one-line header, matching Write’s rules when ansi matches UseANSISplashFor(os.Stderr).
func BannerContent(client core.StreamClient, version, mcpLine string, ansi bool) string {
	if !ansi {
		return plainFallback(client, version, mcpLine)
	}
	return Render(client, version, mcpLine)
}

// Write renders the splash to w using the same ansi vs plain rules as UseANSISplashFor(w).
func Write(w io.Writer, client core.StreamClient, version, mcpLine string) error {
	ansi := UseANSISplashFor(w)
	_, err := fmt.Fprintln(w, BannerContent(client, version, mcpLine, ansi))
	return err
}
