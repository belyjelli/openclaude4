package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/ghstatus"
	"github.com/gitlawb/openclaude4/internal/mcp"
	"github.com/gitlawb/openclaude4/internal/providers"
	"github.com/gitlawb/openclaude4/internal/startupbanner"
	"github.com/gitlawb/openclaude4/internal/tools"
	"github.com/mattn/go-isatty"
)

const doctorReleaseAPI = "https://api.github.com/repos/gitlawb/openclaude4/releases/latest"

// PrintDoctorReport writes the same diagnostics as the doctor subcommand.
func PrintDoctorReport(w io.Writer, ver, cmt string) {
	if w == nil {
		w = io.Discard
	}

	streamClient, clientErr := providers.NewStreamClient()
	bannerClient := streamClient
	if clientErr != nil {
		bannerClient = providers.DoctorBannerClient()
	}

	mcpLine := mcpDoctorSummaryLine()
	ansi := startupbanner.UseANSISplashFor(w)
	shell := shellDisplayName()
	banner := startupbanner.BannerContent(bannerClient, ver, mcpLine, ansi, shell)
	_, _ = fmt.Fprintln(w, banner)

	tty := startupbanner.WriterIsTerminal(w)
	bold, reset := "", ""
	if tty {
		bold = "\x1b[1m"
		reset = "\x1b[0m"
	}

	installKind, installDetail := doctorInstallationKind(ver, cmt)
	wd, _ := os.Getwd()
	if wd == "" {
		wd = "."
	}
	invoked := doctorInvokedBinary()

	_, _ = fmt.Fprintf(w, "%sDiagnostics%s\n", bold, reset)
	_, _ = fmt.Fprintf(w, "└ Currently running: %s (%s)\n", installKind, installDetail)
	_, _ = fmt.Fprintf(w, "└ Path: %s\n", wd)
	_, _ = fmt.Fprintf(w, "└ Invoked: %s\n", invoked)
	_, _ = fmt.Fprintf(w, "└ Config install method: unknown\n")

	if err := config.Validate(); err != nil {
		_, _ = fmt.Fprintf(w, "└ Config validation: %v\n", err)
	}

	if _, err := exec.LookPath("rg"); err != nil {
		_, _ = fmt.Fprintln(w, "└ Search: Not working (Grep tool uses Go regexp only)")
	} else {
		_, _ = fmt.Fprintln(w, "└ Search: OK (system ripgrep on PATH)")
	}

	if _, err := exec.LookPath("gh"); err != nil {
		_, _ = fmt.Fprintln(w, "└ GitHub CLI: not on PATH (install https://cli.github.com/ and run gh auth login for PR/issue workflows)")
	} else {
		_, authed := ghstatus.GhAuthSummary(context.Background())
		if authed {
			_, _ = fmt.Fprintln(w, "└ GitHub CLI: gh on PATH, authenticated (gh auth token OK; local check, no network)")
		} else {
			_, _ = fmt.Fprintln(w, "└ GitHub CLI: gh on PATH, not authenticated — run: gh auth login")
		}
	}

	_, _ = fmt.Fprintf(w, "\n%sUpdates%s\n", bold, reset)
	if installKind == "development" {
		_, _ = fmt.Fprintln(w, "└ Auto-updates: disabled (development build)")
	} else {
		_, _ = fmt.Fprintln(w, "└ Auto-updates: disabled (use your install channel)")
	}
	_, _ = fmt.Fprintln(w, "└ Auto-update channel: latest")
	if tag, ok := tryLatestReleaseTag(); ok {
		_, _ = fmt.Fprintf(w, "└ Latest release: %s\n", tag)
	} else {
		if tty {
			_, _ = fmt.Fprint(w, "\x1b[2m")
		}
		_, _ = fmt.Fprintln(w, "└ Failed to fetch versions")
		if tty {
			_, _ = fmt.Fprint(w, reset)
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Go runtime: %s\n", runtime.Version())

	if _, err := exec.LookPath("spider"); err != nil {
		_, _ = fmt.Fprintf(w, "spider (spider_cli): not found on PATH (optional SpiderScrape tool not registered; cargo install spider_cli)\n")
	} else {
		p, _ := exec.LookPath("spider")
		_, _ = fmt.Fprintf(w, "spider (spider_cli): found at %s — SpiderScrape tool enabled\n", p)
	}

	// PaperCLI: optional; same rules as tools.PaperCLIRegistered (PATH or OPENCLAUDE_PAPERCLI / PAPERCLI_BIN).
	if p, set := os.Getenv("OPENCLAUDE_PAPERCLI"), os.Getenv("PAPERCLI_BIN"); strings.TrimSpace(p) != "" || strings.TrimSpace(set) != "" {
		_, _ = fmt.Fprintf(w, "papercli: OPENCLAUDE_PAPERCLI or PAPERCLI_BIN set — PaperCLI tool enabled\n")
	} else if pp, err := exec.LookPath("papercli"); err != nil {
		_, _ = fmt.Fprintf(w, "papercli: not on PATH (optional PaperCLI tool not registered; build/install papercli)\n")
	} else {
		_, _ = fmt.Fprintf(w, "papercli: found at %s — PaperCLI tool enabled\n", pp)
	}

	if sp, ok := tools.SpeedtestCLIBinary(); ok {
		_, _ = fmt.Fprintf(w, "speedtest-cli (LibreSpeed): found at %s — SpeedtestCLI tool enabled\n", sp)
	} else {
		_, _ = fmt.Fprintf(w, "speedtest-cli (LibreSpeed): not found (optional SpeedtestCLI tool not registered; install librespeed-cli or speedtcli, or set OPENCLAUDE_SPEEDTEST_CLI / SPEEDTEST_CLI_BIN)\n")
	}

	_, _ = fmt.Fprintf(w, "%s\n", providers.PingProviderBestEffort())

	mcpSrv, mcpSrc, _ := mcp.ResolveFromEnvironment()
	if len(mcpSrv) == 0 {
		_, _ = fmt.Fprintln(w, "MCP (effective): no servers configured")
	} else {
		src := "v2"
		if mcpSrc == mcp.SourceLegacy {
			src = "legacy mcp.servers"
		}
		_, _ = fmt.Fprintf(w, "MCP (effective, %s): %d server(s)\n", src, len(mcpSrv))
		for _, s := range mcpSrv {
			cmd0 := ""
			if len(s.Command) > 0 {
				cmd0 = s.Command[0]
			}
			ap := s.Approval
			if ap == "" {
				ap = "ask"
			}
			_, _ = fmt.Fprintf(w, "  - %s: argv0=%q approval=%s\n", s.Name, cmd0, ap)
		}
	}

	if clientErr != nil {
		_, _ = fmt.Fprintf(w, "Client: error — %v\n", clientErr)
	} else {
		_, _ = fmt.Fprintln(w, "Client: configuration OK for chat")
	}

	if tty && isatty.IsTerminal(uintptr(os.Stdin.Fd())) && os.Getenv("CI") == "" &&
		!envTruthy("OPENCLAUDE_DOCTOR_NO_WAIT") {
		_, _ = fmt.Fprintln(w, "\nPress Enter to continue...")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

func mcpDoctorSummaryLine() string {
	cfg, _, err := mcp.ResolveFromEnvironment()
	if err != nil || len(cfg) == 0 {
		return ""
	}
	return fmt.Sprintf("MCP: %d server(s) in config — /mcp list", len(cfg))
}

func doctorInstallationKind(ver, cmt string) (kind, detail string) {
	detail = cmt
	if detail == "" {
		detail = "unknown"
	}
	if strings.Contains(strings.ToLower(ver), "dev") || cmt == "" || cmt == "unknown" {
		return "development", detail
	}
	return "native", detail
}

func doctorInvokedBinary() string {
	ex, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	if r, err := filepath.EvalSymlinks(ex); err == nil {
		return r
	}
	return ex
}

func shellDisplayName() string {
	s := strings.TrimSpace(os.Getenv("SHELL"))
	if s == "" {
		return ""
	}
	base := filepath.Base(s)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	r := []rune(base)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func tryLatestReleaseTag() (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, doctorReleaseAPI, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "openclaude-doctor/1")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return "", false
	}
	tag := strings.TrimSpace(body.TagName)
	if tag == "" {
		return "", false
	}
	return tag, true
}
