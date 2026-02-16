package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "ctx",
	Short: "ContextSubstrate — reproducible, debuggable, contestable AI agent execution",
	Long: `ctx is an execution substrate for AI agents that makes their work
reproducible, debuggable, and contestable using developer-native primitives
(files, hashes, diffs, CLI workflows).`,
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose/debug output")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ctx: %s\n", err)
		os.Exit(1)
	}
}
