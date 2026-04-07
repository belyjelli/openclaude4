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
	Long: `OpenClaude v4 is a native Go implementation. Phase 0: OpenAI-compatible
streaming chat. Set OPENAI_API_KEY. Optional: OPENAI_BASE_URL, OPENAI_MODEL.`,
	RunE: runChat,
}

func init() {
	rootCmd.PersistentFlags().String("model", "", "Chat model (overrides OPENAI_MODEL / default)")
	rootCmd.PersistentFlags().String("base-url", "", "OpenAI-compatible API base URL (overrides OPENAI_BASE_URL)")

	_ = viper.BindPFlag("provider.model", rootCmd.PersistentFlags().Lookup("model"))
	_ = viper.BindPFlag("provider.base_url", rootCmd.PersistentFlags().Lookup("base-url"))

	rootCmd.PersistentPreRun = func(*cobra.Command, []string) {
		config.Load()
	}

	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(*cobra.Command, []string) {
		fmt.Fprintln(os.Stdout, "openclaude", version, "("+commit+")")
	},
}

var (
	version = "0.0.0-dev"
	commit  = "unknown"
)
