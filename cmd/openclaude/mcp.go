package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
  openclaude mcp add     — append a server entry (see mcp add --help; --bunx for npm MCP via bunx -y)`),
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

var mcpAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Append an MCP stdio server to your openclaude config file",
	Long: strings.TrimSpace(`
Writes under mcp.servers in the same config file WritableConfigPath picks (see docs/CONFIG.md):
--config path when set, else ./openclaude.{yaml,yml,json} if it exists, else ~/.config/openclaude/openclaudev4.* if it exists, else ~/.config/openclaude/openclaude.*, else a new ~/.config/openclaude/openclaude.yaml.

Repeat --exec for each argv token, in order.

Recommended for npm MCP packages (Bun installs/runs the package, like npx):

  openclaude mcp add --name fs --bunx --exec @modelcontextprotocol/server-filesystem --exec /tmp

Equivalent without --bunx:

  openclaude mcp add --name fs --exec bunx --exec -y --exec @modelcontextprotocol/server-filesystem --exec /tmp

Local script (explicit argv):

  openclaude mcp add --name mine --exec bun --exec run --exec ./mcp-server.ts

YAML comments are not preserved on rewrite. Duplicate server names in that file are rejected.`),
	RunE: runMCPAdd,
}

func init() {
	mcpAddCmd.Flags().String("name", "", "Unique server name (required)")
	mcpAddCmd.Flags().String("approval", "ask", "Tool approval: ask | always | never")
	mcpAddCmd.Flags().StringSlice("exec", nil, "One command argv token; repeat in order (required)")
	mcpAddCmd.Flags().Bool("bunx", false, "Prepend bunx -y to the command (recommended for npm MCP packages; still pass --exec for the package id and arguments)")
	mcpAddCmd.Flags().Bool("dry-run", false, "Print target path and entry without writing")
	_ = mcpAddCmd.MarkFlagRequired("name")

	mcpCmd.AddCommand(mcpListCmd, mcpDoctorCmd, mcpAddCmd)
	rootCmd.AddCommand(mcpCmd)
}

func runMCPList(_ *cobra.Command, _ []string) error {
	PrintMCPConfigList(os.Stdout)
	return nil
}

// PrintMCPConfigList prints MCP servers as defined in config (no subprocesses).
func PrintMCPConfigList(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	srv := config.MCPServers()
	if len(srv) == 0 {
		_, _ = fmt.Fprintln(w, "(no MCP servers in mcp.servers — see docs/CONFIG.md)")
		return
	}
	_, _ = fmt.Fprintf(w, "%d MCP server(s) in config:\n", len(srv))
	for _, s := range srv {
		ap := strings.TrimSpace(s.Approval)
		if ap == "" {
			ap = "ask"
		}
		_, _ = fmt.Fprintf(w, "\n- name: %s\n  approval: %s\n", s.Name, ap)
		if len(s.Command) > 0 {
			_, _ = fmt.Fprintf(w, "  command: %q\n", s.Command)
		}
		if len(s.Env) > 0 {
			keys := make([]string, 0, len(s.Env))
			for k := range s.Env {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			_, _ = fmt.Fprintln(w, "  env:")
			for _, k := range keys {
				_, _ = fmt.Fprintf(w, "    %s: %q\n", k, s.Env[k])
			}
		}
	}
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

// mcpAddCommandArgv builds the subprocess argv for mcp add: optional bunx -y prefix plus --exec tokens.
func mcpAddCommandArgv(bunx bool, execParts []string) []string {
	var out []string
	if bunx {
		out = append(out, "bunx", "-y")
	}
	for _, p := range execParts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func runMCPAdd(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	approval, _ := cmd.Flags().GetString("approval")
	execParts, _ := cmd.Flags().GetStringSlice("exec")
	bunx, _ := cmd.Flags().GetBool("bunx")
	dry, _ := cmd.Flags().GetBool("dry-run")

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	var userExec []string
	for _, p := range execParts {
		p = strings.TrimSpace(p)
		if p != "" {
			userExec = append(userExec, p)
		}
	}
	if bunx && len(userExec) == 0 {
		return fmt.Errorf("--bunx requires at least one --exec (npm package name and optional arguments)")
	}
	cmdParts := mcpAddCommandArgv(bunx, userExec)
	if len(cmdParts) == 0 {
		return fmt.Errorf("at least one non-empty --exec token is required")
	}

	path, err := config.WritableConfigPath()
	if err != nil {
		return err
	}

	srv := config.MCPServer{Name: name, Command: cmdParts, Approval: approval}
	if dry {
		_, _ = fmt.Fprintf(os.Stdout, "Would append to %s:\n  name: %s\n  approval: %s\n  command: %q\n", path, srv.Name, config.NormalizeMCPApproval(approval), srv.Command)
		return nil
	}

	if err := config.AppendMCPServerToConfigFile(path, srv); err != nil {
		if errors.Is(err, config.ErrMCPNameExists) {
			return fmt.Errorf("%w (use a different --name or edit the file)", err)
		}
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "Appended MCP server %q to %s\n", name, path)
	return nil
}
