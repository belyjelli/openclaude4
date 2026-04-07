package main

import (
	"fmt"
	"os"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "openclaude",
	Short: "OpenClaude v4 — terminal coding agent (Go rewrite)",
	Long: `OpenClaude v4 is a native Go implementation: multi-provider chat with tools.

Config: env vars, optional openclaude.yaml, optional v3 .openclaude-profile.json (see docs/CONFIG.md). Examples:
  OPENAI_API_KEY, OPENAI_BASE_URL, OPENAI_MODEL
  OPENCLAUDE_PROVIDER=ollama|gemini, OLLAMA_*, GEMINI_API_KEY / GOOGLE_API_KEY`,
	RunE: runChat,
}

func init() {
	config.InitSessionDefaults()

	rootCmd.PersistentFlags().String("config", "", "Path to config file (yaml/json); overrides default search paths")
	rootCmd.PersistentFlags().String("provider", "", "Provider: openai, ollama, or gemini (overrides OPENCLAUDE_PROVIDER)")
	rootCmd.PersistentFlags().String("model", "", "Chat model (provider-specific default if empty)")
	rootCmd.PersistentFlags().String("base-url", "", "OpenAI-compatible API base URL (OpenAI provider only)")

	_ = viper.BindPFlag("provider.model", rootCmd.PersistentFlags().Lookup("model"))
	_ = viper.BindPFlag("provider.base_url", rootCmd.PersistentFlags().Lookup("base-url"))
	_ = viper.BindPFlag("provider.name", rootCmd.PersistentFlags().Lookup("provider"))

	rootCmd.PersistentFlags().String("session", "", "Session id for on-disk transcript (default: new id each run; env OPENCLAUDE_SESSION)")
	_ = viper.BindPFlag("session.name", rootCmd.PersistentFlags().Lookup("session"))
	rootCmd.PersistentFlags().Bool("resume", false, "Resume the last saved session (env OPENCLAUDE_RESUME=true)")
	_ = viper.BindPFlag("session.resume_last", rootCmd.PersistentFlags().Lookup("resume"))
	rootCmd.PersistentFlags().Bool("list-sessions", false, "List saved sessions on disk and exit (no API key required)")
	rootCmd.PersistentFlags().Bool("no-session", false, "Disable on-disk session persistence")
	_ = viper.BindPFlag("session.disabled", rootCmd.PersistentFlags().Lookup("no-session"))

	rootCmd.PersistentFlags().Bool("tui", false, "Full-screen Bubble Tea UI (kernel events: streaming, tools, permissions)")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		path, _ := cmd.Flags().GetString("config")
		config.Load(path)
	}

	rootCmd.AddCommand(versionCmd, doctorCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(*cobra.Command, []string) {
		_, _ = fmt.Fprintln(os.Stdout, "openclaude", version, "("+commit+")")
	},
}

var (
	version = "0.0.0-dev"
	commit  = "unknown"
)
