package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/contextsubstrate/ctx/internal/delta"
	"github.com/contextsubstrate/ctx/internal/graph"
	ctxdiff "github.com/contextsubstrate/ctx/internal/diff"
	"github.com/contextsubstrate/ctx/internal/index"
	"github.com/contextsubstrate/ctx/internal/optimize"
	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/replay"
	"github.com/contextsubstrate/ctx/internal/sharing"
	"github.com/contextsubstrate/ctx/internal/store"
	"github.com/contextsubstrate/ctx/internal/telemetry"
	"github.com/contextsubstrate/ctx/internal/verify"
	"github.com/spf13/cobra"
)

var diffHuman bool
var indexCommit string
var deltaBase string
var deltaHead string
var deltaHuman bool
var optimizeTask string
var optimizeCommit string
var optimizeTokenCap int
var optimizeIncludeTests bool
var optimizeHuman bool
var metricsLimit int
var benchmarkCommits int

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

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index a commit into the context graph",
	Long:  "Build the JSONL context graph for a git commit. Indexes file snapshots, path records, and commit metadata.",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		repoRoot, err := index.GetRepoRoot(dir)
		if err != nil {
			return err
		}

		commitSHA := indexCommit
		if commitSHA == "" {
			commitSHA, err = index.GetHeadSHA(repoRoot)
			if err != nil {
				return err
			}
		}

		if err := index.IndexCommit(root, repoRoot, commitSHA); err != nil {
			return fmt.Errorf("indexing commit: %w", err)
		}

		fmt.Printf("Indexed commit %s\n", commitSHA[:8])
		return nil
	},
}

var deltaCmd = &cobra.Command{
	Use:   "delta",
	Short: "Show changes between two indexed commits",
	Long:  "Compare two indexed commits and report file-level changes. Both commits must be previously indexed.",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		if deltaBase == "" || deltaHead == "" {
			return fmt.Errorf("both --base and --head flags are required")
		}

		report, err := delta.ComputeDelta(root, deltaBase, deltaHead)
		if err != nil {
			return err
		}

		if deltaHuman {
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

var optimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Generate an optimized context pack for a task",
	Long:  "Select the most relevant files and symbols for a task within a token budget, using the indexed context graph.",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		if optimizeTask == "" {
			return fmt.Errorf("--task flag is required")
		}

		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		repoRoot, err := index.GetRepoRoot(dir)
		if err != nil {
			return err
		}

		req := &optimize.PackRequest{
			Commit:       optimizeCommit,
			Task:         optimizeTask,
			TokenCap:     optimizeTokenCap,
			IncludeTests: optimizeIncludeTests,
		}

		result, err := optimize.GeneratePack(root, repoRoot, req)
		if err != nil {
			return err
		}

		if optimizeHuman {
			fmt.Print(result.Human())
		} else {
			data, err := result.JSON()
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		}
		return nil
	},
}

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display token savings dashboard",
	Long:  "Show token optimization metrics including savings per run, cache hit rates, and ROI summary.",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		metrics, err := telemetry.GetMetrics(root, metricsLimit)
		if err != nil {
			return err
		}

		roi := telemetry.ComputeROI(metrics)
		fmt.Print(telemetry.FormatMetrics(metrics, roi))
		return nil
	},
}

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Compare cold vs warm context across commits",
	Long:  "Simulate cold (full repo scan) vs warm (incremental delta) token usage across recent commits to measure optimization gains.",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := store.DiscoverStore()
		if err != nil {
			return err
		}

		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		repoRoot, err := index.GetRepoRoot(dir)
		if err != nil {
			return err
		}

		headSHA, err := index.GetHeadSHA(repoRoot)
		if err != nil {
			return err
		}

		// Get recent commits
		commits, err := getRecentCommits(repoRoot, benchmarkCommits)
		if err != nil {
			return err
		}

		if len(commits) < 2 {
			return fmt.Errorf("need at least 2 commits for benchmarking, got %d", len(commits))
		}

		fmt.Printf("Benchmarking %d commits (HEAD: %s)\n", len(commits), headSHA[:8])
		fmt.Println("═══════════════════════════════════════")

		// Index all commits
		for _, sha := range commits {
			if err := index.IndexCommit(root, repoRoot, sha); err != nil {
				return fmt.Errorf("indexing %s: %w", sha[:8], err)
			}
		}

		// Compare cold vs warm for each pair
		fmt.Printf("\n%-10s  %10s  %10s  %10s  %6s\n", "Commit", "Cold (est)", "Warm (est)", "Saved", "Pct")
		fmt.Printf("%-10s  %10s  %10s  %10s  %6s\n", "──────", "──────────", "──────────", "─────", "───")

		for i := 1; i < len(commits); i++ {
			baseSHA := commits[i-1]
			headSHA := commits[i]

			cold, err := telemetry.EstimateBaseline(root, headSHA)
			if err != nil {
				continue
			}

			deltaReport, err := delta.ComputeDelta(root, baseSHA, headSHA)
			if err != nil {
				continue
			}

			// Estimate warm tokens from changed files only
			warm := estimateWarmTokens(root, headSHA, deltaReport)
			saved := cold - warm
			pct := 0.0
			if cold > 0 {
				pct = float64(saved) / float64(cold) * 100
			}

			fmt.Printf("%-10s  %10d  %10d  %10d  %5.1f%%\n",
				headSHA[:8], cold, warm, saved, pct)
		}

		return nil
	},
}

// getRecentCommits returns the N most recent commits.
func getRecentCommits(repoRoot string, n int) ([]string, error) {
	cmd := exec.Command("git", "log", "--format=%H", fmt.Sprintf("-%d", n))
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	var commits []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			commits = append(commits, line)
		}
	}

	// Reverse to get oldest first
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}

	return commits, nil
}

func estimateWarmTokens(storeRoot, headSHA string, dr *delta.DeltaReport) int {
	files, err := graph.ReadRecords[graph.FileSnapshot](graph.FilesPath(storeRoot, headSHA))
	if err != nil {
		return 0
	}

	changedSet := make(map[string]bool)
	for _, f := range dr.FilesChanged {
		changedSet[f] = true
	}
	for _, f := range dr.FilesAdded {
		changedSet[f] = true
	}

	paths, _ := graph.ReadRecords[graph.PathRecord](graph.PathsPath(storeRoot))
	pathLookup := make(map[string]string)
	for _, p := range paths {
		pathLookup[p.PathID] = p.Path
	}

	tokens := 0
	for _, f := range files {
		path := pathLookup[f.PathID]
		if changedSet[path] && !f.IsBinary && !f.IsGenerated {
			tokens += int(float64(f.ByteSize) * 0.25)
		}
	}

	return tokens
}

func init() {
	diffCmd.Flags().BoolVar(&diffHuman, "human", false, "output human-readable summary instead of JSON")
	indexCmd.Flags().StringVar(&indexCommit, "commit", "", "specific commit SHA to index (defaults to HEAD)")
	deltaCmd.Flags().StringVar(&deltaBase, "base", "", "base commit SHA (required)")
	deltaCmd.Flags().StringVar(&deltaHead, "head", "", "head commit SHA (required)")
	deltaCmd.Flags().BoolVar(&deltaHuman, "human", false, "output human-readable summary instead of JSON")
	optimizeCmd.Flags().StringVar(&optimizeTask, "task", "", "task description for context selection (required)")
	optimizeCmd.Flags().StringVar(&optimizeCommit, "commit", "", "commit SHA to generate pack for (defaults to HEAD)")
	optimizeCmd.Flags().IntVar(&optimizeTokenCap, "token-cap", optimize.DefaultTokenCap, "maximum token budget")
	optimizeCmd.Flags().BoolVar(&optimizeIncludeTests, "include-tests", false, "include test files in the pack")
	optimizeCmd.Flags().BoolVar(&optimizeHuman, "human", false, "output human-readable summary instead of JSON")
	metricsCmd.Flags().IntVar(&metricsLimit, "limit", 20, "number of recent runs to display")
	benchmarkCmd.Flags().IntVar(&benchmarkCommits, "commits", 10, "number of recent commits to benchmark")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(packCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(replayCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(forkCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(deltaCmd)
	rootCmd.AddCommand(optimizeCmd)
	rootCmd.AddCommand(metricsCmd)
	rootCmd.AddCommand(benchmarkCmd)

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
