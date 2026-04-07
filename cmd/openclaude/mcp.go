package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/tools"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Inspect configured Model Context Protocol (stdio) servers",
	Long: strings.TrimSpace(`
MCP servers are defined under mcp.servers in openclaude.yaml (see docs/CONFIG.md).

  openclaude mcp list    — print configured servers from config (no subprocesses)
  openclaude mcp doctor  — connect to each server, list tools (like chat startup)

To add a server, edit the config file; there is no mcp add subcommand yet.`),
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "Print MCP servers from config (does not start subprocesses)",
	RunE:  runMCPList,
}

var mcpDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Connect to each configured MCP server and list tools",
	Long: strings.TrimSpace(`
Spawns each configured stdio server, negotiates MCP, and lists tools (same path as the REPL).
Failures are reported on stderr; exit status 1 if any server fails or if configured servers exist but none connect.`),
	RunE: runMCPDoctor,
}

func init() {
	mcpCmd.AddCommand(mcpListCmd, mcpDoctorCmd)
	rootCmd.AddCommand(mcpCmd)
}

func runMCPList(_ *cobra.Command, _ []string) error {
	srv := config.MCPServers()
	if len(srv) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "(no MCP servers in mcp.servers — see docs/CONFIG.md)")
		return nil
	}
	_, _ = fmt.Fprintf(os.Stdout, "%d MCP server(s) in config:\n", len(srv))
	for _, s := range srv {
		ap := strings.TrimSpace(s.Approval)
		if ap == "" {
			ap = "ask"
		}
		_, _ = fmt.Fprintf(os.Stdout, "\n- name: %s\n  approval: %s\n", s.Name, ap)
		if len(s.Command) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "  command: %q\n", s.Command)
		}
		if len(s.Env) > 0 {
			keys := make([]string, 0, len(s.Env))
			for k := range s.Env {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			_, _ = fmt.Fprintln(os.Stdout, "  env:")
			for _, k := range keys {
				_, _ = fmt.Fprintf(os.Stdout, "    %s: %q\n", k, s.Env[k])
			}
		}
	}
	return nil
}

func runMCPDoctor(_ *cobra.Command, _ []string) error {
	cfg := config.MCPServers()
	if len(cfg) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "(no MCP servers configured — nothing to check)")
		return nil
	}

	ctx := context.Background()
	reg := tools.NewRegistry()
	mgr := mcpclient.ConnectAndRegister(ctx, reg, cfg, os.Stderr)
	defer mgr.Close()

	_, _ = fmt.Fprintln(os.Stdout, mgr.DescribeServers())

	n := len(mgr.Servers)
	if n < len(cfg) {
		_, _ = fmt.Fprintf(os.Stderr, "openclaude mcp doctor: %d of %d server(s) connected (see stderr above for errors)\n", n, len(cfg))
		return fmt.Errorf("mcp doctor: %d server(s) failed", len(cfg)-n)
	}
	_, _ = fmt.Fprintf(os.Stdout, "\nOK: all %d configured server(s) connected.\n", n)
	return nil
}
