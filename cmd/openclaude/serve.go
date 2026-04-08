package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	ocrpc "github.com/gitlawb/openclaude4/internal/grpc"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/providers"
	"github.com/gitlawb/openclaude4/internal/providers/openaicomp"
	"github.com/gitlawb/openclaude4/internal/skills"
	"github.com/gitlawb/openclaude4/internal/tools"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the OpenClaude gRPC API (same kernel as chat/TUI)",
	Long: `Loads the same config as the REPL, listens for gRPC, and serves openclaude.v4.AgentService.

Listen address: --listen, or env OPENCLAUDE_GRPC_ADDR, or default :50051.
Session files use the same rules as the REPL (see docs/CONFIG.md); ChatRequest.session_id binds a stream to an on-disk session when persistence is enabled.`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().String("listen", "", "gRPC listen address (overrides OPENCLAUDE_GRPC_ADDR)")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, _ []string) error {
	if err := config.Validate(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}

	client, err := providers.NewStreamClient()
	if err != nil {
		switch {
		case errors.Is(err, openaicomp.ErrMissingAPIKey):
			_, _ = fmt.Fprintln(os.Stderr, "Error: set OPENAI_API_KEY (or use --provider ollama / gemini as appropriate).")
			return err
		case errors.Is(err, openaicomp.ErrMissingGeminiKey):
			_, _ = fmt.Fprintln(os.Stderr, "Error: set GEMINI_API_KEY or GOOGLE_API_KEY for provider gemini.")
			return err
		case errors.Is(err, openaicomp.ErrMissingGitHubToken):
			_, _ = fmt.Fprintln(os.Stderr, "Error: set GITHUB_TOKEN or GITHUB_PAT for provider github.")
			return err
		case errors.Is(err, providers.ErrCodexNotImplemented):
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		return err
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	wd, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	ctx = tools.WithWorkDir(ctx, wd)

	skillCat, err := skills.Load(config.SkillDirs())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "openclaude: skills: %v (continuing without skills)\n", err)
		skillCat = skills.EmptyCatalog()
	}
	reg := tools.NewDefaultRegistry(skillCat)
	mcpMgr := mcpclient.ConnectAndRegister(ctx, reg, config.MCPServers(), os.Stderr)
	defer mcpMgr.Close()

	var taskSlot atomic.Pointer[core.Agent]
	reg.Register(core.NewTaskTool(func() *core.Agent {
		return taskSlot.Load()
	}))

	autoApprove := strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "1") ||
		strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "true")

	addr, _ := cmd.Flags().GetString("listen")
	if strings.TrimSpace(addr) == "" {
		addr = strings.TrimSpace(os.Getenv("OPENCLAUDE_GRPC_ADDR"))
	}
	if addr == "" {
		addr = ":50051"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	_, _ = fmt.Fprintf(os.Stderr, "openclaude gRPC listening on %s (openclaude.v4.AgentService)\n", ln.Addr().String())

	gs := grpc.NewServer()
	k := ocrpc.Kernel{
		Client:      client,
		Registry:    reg,
		AutoApprove: autoApprove,
		TaskParent:  &taskSlot,
		Session: ocrpc.SessionOpts{
			Disabled: config.SessionDisabled(),
			Dir:      config.EffectiveSessionDir(),
		},
	}
	ocrpc.Register(gs, k)

	if err := gs.Serve(ln); err != nil {
		return err
	}
	return nil
}
