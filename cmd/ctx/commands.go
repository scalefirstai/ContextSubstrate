package main

import (
	"fmt"
	"os"

	ctxdiff "github.com/contextsubstrate/ctx/internal/diff"
	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/replay"
	"github.com/contextsubstrate/ctx/internal/sharing"
	"github.com/contextsubstrate/ctx/internal/store"
	"github.com/contextsubstrate/ctx/internal/verify"
	"github.com/spf13/cobra"
)

var diffHuman bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new context store",
	Long:  "Create a .ctx/ directory in the current working directory with the required subdirectory structure.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		root, err := store.InitStore(dir)
		if err != nil {
			return err
		}
		fmt.Printf("Initialized context store at %s\n", root)
		return nil
	},
}

var packCmd = &cobra.Command{
	Use:   "pack <log-file>",
	Short: "Create a context pack from an execution log",
	Long:  "Read an execution log (JSON) and produce an immutable, content-addressed Context Pack.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		log, err := pack.ParseExecutionLog(args[0])
		if err != nil {
			return err
		}

		p, err := pack.CreatePack(root, log)
		if err != nil {
			return err
		}

		if err := pack.RegisterPack(root, p.Hash); err != nil {
			return fmt.Errorf("registering pack: %w", err)
		}

		_, hex, _ := store.ParseHash(p.Hash)
		fmt.Printf("ctx://%s\n", hex)
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <hash>",
	Short: "Inspect a context pack",
	Long:  "Display the contents of a context pack in human-readable format.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		p, err := pack.LoadPack(root, args[0])
		if err != nil {
			return err
		}

		fmt.Print(pack.FormatPack(p))
		return nil
	},
}

var replayCmd = &cobra.Command{
	Use:   "replay <hash>",
	Short: "Replay a captured agent run",
	Long:  "Re-execute an agent run step-by-step as recorded in the context pack.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		report, err := replay.Replay(root, args[0])
		if err != nil {
			return err
		}

		fmt.Print(report.Summary())

		switch report.Fidelity {
		case replay.FidelityDegraded:
			os.Exit(1)
		case replay.FidelityFailed:
			os.Exit(2)
		}
		return nil
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff <hash-a> <hash-b>",
	Short: "Compare two context packs",
	Long:  "Compare two context packs and produce a structured drift report.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		report, err := ctxdiff.Diff(root, args[0], args[1])
		if err != nil {
			return err
		}

		if diffHuman {
			fmt.Print(report.Human())
		} else {
			data, err := report.JSON()
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		}
		return nil
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify <artifact>",
	Short: "Verify artifact provenance",
	Long:  "Check an artifact's provenance by validating its sidecar metadata against the context store.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		result, err := verify.Verify(root, args[0])
		if err != nil {
			return err
		}

		fmt.Print(verify.FormatVerifyResult(result))
		return nil
	},
}

var forkCmd = &cobra.Command{
	Use:   "fork <hash>",
	Short: "Fork a context pack",
	Long:  "Create a new mutable draft derived from an existing context pack.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		draftPath, err := sharing.Fork(root, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Draft created at %s\n", draftPath)
		fmt.Println("Edit the draft, then finalize with 'ctx finalize <draft-path>'")
		return nil
	},
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "List context packs",
	Long:  "List all finalized context packs in the store, ordered by creation date.",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		summaries, err := sharing.ListPacks(root, 50)
		if err != nil {
			return err
		}

		fmt.Print(sharing.FormatPackList(summaries))
		return nil
	},
}

func init() {
	diffCmd.Flags().BoolVar(&diffHuman, "human", false, "output human-readable summary instead of JSON")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(packCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(replayCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(forkCmd)
	rootCmd.AddCommand(logCmd)

	// Shell completion (bash, zsh, fish, powershell) via Cobra's built-in generator
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate a shell completion script for ctx.

To load completions:

Bash:
  $ source <(ctx completion bash)

Zsh:
  $ ctx completion zsh > "${fpath[1]}/_ctx"

Fish:
  $ ctx completion fish | source

PowerShell:
  PS> ctx completion powershell | Out-String | Invoke-Expression`,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	rootCmd.AddCommand(completionCmd)
}
