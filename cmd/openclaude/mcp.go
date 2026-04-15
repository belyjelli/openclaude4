package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/mcp"
	"github.com/gitlawb/openclaude4/internal/mcp/installer"
	mcpv2 "github.com/gitlawb/openclaude4/internal/mcp/v2"
	"github.com/gitlawb/openclaude4/internal/tools"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Inspect and configure Model Context Protocol servers (v2 + legacy)",
	Long: strings.TrimSpace(`
MCP v2 uses ~/.openclaude/mcp.yaml and .mcp.v2.yaml in the project tree (version: "2").
Legacy mcp.servers in openclaude.yaml is still read when no v2 servers are configured, but is deprecated.

  openclaude mcp list     — effective servers (v2 cascade + legacy fallback)
  openclaude mcp doctor   — connect and list tools
  openclaude mcp add      — append a stdio server to ./.mcp.v2.yaml
  openclaude mcp install  — smart install from a GitHub URL
  openclaude mcp migrate  — copy legacy mcp.servers to ~/.openclaude/mcp.yaml`),
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "Print effective MCP servers from the v2 cascade (and legacy fallback)",
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
	Short: "Append an MCP stdio server to ./.mcp.v2.yaml in the current directory",
	Long: strings.TrimSpace(`
Creates or updates .mcp.v2.yaml (MCP v2) in the working directory.

Repeat --exec for each argv token, in order.

Recommended for npm MCP packages (Bun installs/runs the package, like npx):

  openclaude mcp add --name fs --bunx --exec @modelcontextprotocol/server-filesystem --exec /tmp

Equivalent without --bunx:

  openclaude mcp add --name fs --exec npx --exec -y --exec @modelcontextprotocol/server-filesystem --exec /tmp

YAML comments are not preserved on rewrite. Duplicate server names in that file are rejected.`),
	RunE: runMCPAdd,
}

var mcpInstallCmd = &cobra.Command{
	Use:   "install <github-url>",
	Short: "Detect MCP server settings from a public GitHub repo and add to MCP v2 config",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPInstall,
}

var mcpMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Copy legacy mcp.servers from openclaude.yaml into ~/.openclaude/mcp.yaml (MCP v2)",
	RunE:  runMCPMigrate,
}

func init() {
	mcpAddCmd.Flags().String("name", "", "Unique server name (required)")
	mcpAddCmd.Flags().String("approval", "ask", "Tool approval: ask | always | never")
	mcpAddCmd.Flags().StringSlice("exec", nil, "One command argv token; repeat in order (required)")
	mcpAddCmd.Flags().Bool("bunx", false, "Prepend bunx -y to the command (recommended for npm MCP packages; still pass --exec for the package id and arguments)")
	mcpAddCmd.Flags().Bool("dry-run", false, "Print target path and entry without writing")
	_ = mcpAddCmd.MarkFlagRequired("name")

	mcpInstallCmd.Flags().Bool("yes", false, "Non-interactive: pick the highest-confidence candidate")
	mcpInstallCmd.Flags().String("name", "", "Override MCP server name in config")
	mcpInstallCmd.Flags().String("target", "project", "Where to write: project (.mcp.v2.yaml in cwd) or user (~/.openclaude/mcp.yaml)")
	mcpInstallCmd.Flags().Int("pick", 0, "1-based candidate index (default: 1 when --yes)")

	mcpMigrateCmd.Flags().Bool("force", false, "Overwrite non-empty ~/.openclaude/mcp.yaml")

	mcpCmd.AddCommand(mcpListCmd, mcpDoctorCmd, mcpAddCmd, mcpInstallCmd, mcpMigrateCmd)
	rootCmd.AddCommand(mcpCmd)
}

func effectiveMCPContext() (wd string, servers []mcp.ServerConfig, src mcp.ResolveSource, err error) {
	servers, src, err = mcp.ResolveFromEnvironment()
	if err != nil {
		return "", nil, src, err
	}
	wd, _ = os.Getwd()
	return wd, servers, src, nil
}

func runMCPList(_ *cobra.Command, _ []string) error {
	PrintMCPConfigList(os.Stdout)
	return nil
}

// PrintMCPConfigList prints MCP servers from the effective v2 cascade and legacy fallback.
func PrintMCPConfigList(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	_, srv, src, err := effectiveMCPContext()
	if err != nil {
		_, _ = fmt.Fprintf(w, "(error resolving cwd: %v)\n", err)
		return
	}
	if len(srv) == 0 {
		_, _ = fmt.Fprintln(w, "(no MCP servers — configure .mcp.v2.yaml, ~/.openclaude/mcp.yaml, or legacy mcp.servers; see docs/CONFIG.md)")
		return
	}
	switch src {
	case mcp.SourceV2:
		_, _ = fmt.Fprintf(w, "%d MCP server(s) from MCP v2 config:\n", len(srv))
	case mcp.SourceLegacy:
		_, _ = fmt.Fprintf(w, "%d MCP server(s) from legacy openclaude.yaml mcp.servers (deprecated):\n", len(srv))
	}
	for _, s := range srv {
		ap := strings.TrimSpace(s.Approval)
		if ap == "" {
			ap = "ask"
		}
		_, _ = fmt.Fprintf(w, "\n- name: %s\n  transport: %s\n  approval: %s\n", s.Name, strings.TrimSpace(s.Transport), ap)
		if s.ConfigPath != "" {
			_, _ = fmt.Fprintf(w, "  source: %s\n", s.ConfigPath)
		}
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
	if src == mcp.SourceLegacy {
		_, _ = fmt.Fprintln(w, "\nRun: openclaude mcp migrate  — to copy these entries into ~/.openclaude/mcp.yaml (MCP v2).")
	}
}

func runMCPDoctor(_ *cobra.Command, _ []string) error {
	_, cfg, _, err := effectiveMCPContext()
	if err != nil {
		return err
	}
	if len(cfg) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "(no MCP servers configured — nothing to check)")
		return nil
	}

	ctx := context.Background()
	reg := tools.NewRegistry()
	mgr := mcp.ConnectAndRegister(ctx, reg, cfg, os.Stderr)
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

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(wd, ".mcp.v2.yaml")

	srv := mcpv2.Server{
		Name:      name,
		Transport: "stdio",
		Command:   cmdParts,
		Approval:  approval,
	}
	if dry {
		_, _ = fmt.Fprintf(os.Stdout, "Would append to %s:\n  name: %s\n  approval: %s\n  command: %q\n", path, srv.Name, config.NormalizeMCPApproval(approval), srv.Command)
		return nil
	}

	if err := mcpv2.AppendServer(path, srv); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "Appended MCP server %q to %s\n", name, path)
	return nil
}

func runMCPInstall(cmd *cobra.Command, args []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	nameOverride, _ := cmd.Flags().GetString("name")
	target, _ := cmd.Flags().GetString("target")
	pick, _ := cmd.Flags().GetInt("pick")

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	req := installer.InstallRequest{URL: strings.TrimSpace(args[0]), SuggestedName: strings.TrimSpace(nameOverride)}
	meta, cands, err := installer.ParseGitHubRepo(ctx, installer.NewHTTPClient(), req)
	if err != nil {
		return err
	}
	if len(cands) == 0 {
		return fmt.Errorf("no install candidates for %s", req.URL)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Repository: %s/%s @ %s\n", meta.Owner, meta.Repo, meta.Ref)
	for i, c := range cands {
		_, _ = fmt.Fprintf(os.Stdout, "\n[%d] confidence=%.0f from=%s\n    %s\n    command: %q\n", i+1, c.Confidence, c.DetectedFrom, c.Reason, c.Command)
	}

	idx := 0
	if yes {
		if pick > 0 {
			idx = pick - 1
		}
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "\nInstall which candidate? [1-%d] (default 1): ", len(cands))
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line != "" {
			var n int
			_, _ = fmt.Sscanf(line, "%d", &n)
			if n > 0 {
				idx = n - 1
			}
		}
	}
	if idx < 0 || idx >= len(cands) {
		return fmt.Errorf("candidate index out of range")
	}
	chosen := cands[idx]
	if nameOverride != "" {
		chosen.Name = strings.TrimSpace(nameOverride)
	}
	if chosen.Name == "" {
		return fmt.Errorf("server name is empty (use --name)")
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSelected:\n  name: %s\n  command: %q\n", chosen.Name, chosen.Command)
	if !yes {
		_, _ = fmt.Fprint(os.Stdout, "Proceed? [y/N]: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return err
		}
		if strings.ToLower(strings.TrimSpace(line)) != "y" && strings.ToLower(strings.TrimSpace(line)) != "yes" {
			return fmt.Errorf("cancelled")
		}
	}

	var outPath string
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		outPath = filepath.Join(home, ".openclaude", "mcp.yaml")
	case "project", "":
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		outPath = filepath.Join(wd, ".mcp.v2.yaml")
	default:
		return fmt.Errorf("unknown --target %q (project|user)", target)
	}

	srv := mcpv2.Server{
		Name:      chosen.Name,
		Transport: chosen.Transport,
		Command:   append([]string(nil), chosen.Command...),
		Env:       chosen.Env,
		Approval:  config.NormalizeMCPApproval(chosen.Approval),
	}
	if strings.TrimSpace(srv.Transport) == "" {
		srv.Transport = "stdio"
	}
	if err := mcpv2.AppendServer(outPath, srv); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "Wrote MCP server %q to %s\n", srv.Name, outPath)
	return nil
}

func runMCPMigrate(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")
	srv := config.MCPServers()
	if len(srv) == 0 {
		return fmt.Errorf("no legacy mcp.servers in openclaude config (nothing to migrate)")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	wpath, err := config.WritableConfigPath()
	if err != nil {
		return err
	}
	backup := filepath.Join(filepath.Dir(wpath), "mcp.v1.backup.yaml")
	if err := mcp.WriteMCPV1Backup(backup, srv); err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	out, err := mcp.MigrateLegacyToUserV2(home, srv, force)
	if err != nil {
		if errors.Is(err, mcp.ErrMigrateNeedForce) {
			return fmt.Errorf("%w", mcp.ErrMigrateNeedForce)
		}
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "Archived legacy MCP entries to %s\nWrote MCP v2 config to %s\n", backup, out)
	return nil
}
