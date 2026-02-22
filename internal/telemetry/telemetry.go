package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/contextsubstrate/ctx/internal/graph"
)

// TelemetryDir is the subdirectory name within .ctx/ for telemetry data.
const TelemetryDir = "telemetry"

// Run represents a single agent execution run.
type Run struct {
	RunID      string    `json:"run_id"`
	Repo       string    `json:"repo"`
	BaseCommit string    `json:"base_commit"`
	HeadCommit string    `json:"head_commit"`
	Agent      string    `json:"agent"`
	TaskHash   string    `json:"task_hash"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
}

// RunMetrics captures token usage and performance data for a run.
type RunMetrics struct {
	RunID              string  `json:"run_id"`
	BaselineEstTokens  int     `json:"baseline_est_tokens"`
	DeltaTokens        int     `json:"delta_tokens"`
	TokensSaved        int     `json:"tokens_saved"`
	SavingsPct         float64 `json:"savings_pct"`
	CacheHitRate       float64 `json:"cache_hit_rate"`
	FilesInvalidated   int     `json:"files_invalidated"`
	SymbolsInvalidated int     `json:"symbols_invalidated"`
	LatencyMS          int     `json:"latency_ms"`
}

// ROISummary aggregates metrics across multiple runs.
type ROISummary struct {
	TotalRuns         int     `json:"total_runs"`
	TotalBaseline     int     `json:"total_baseline_tokens"`
	TotalDelta        int     `json:"total_delta_tokens"`
	TotalSaved        int     `json:"total_tokens_saved"`
	AvgSavingsPct     float64 `json:"avg_savings_pct"`
	AvgCacheHitRate   float64 `json:"avg_cache_hit_rate"`
	AvgLatencyMS      float64 `json:"avg_latency_ms"`
	BestSavingsPct    float64 `json:"best_savings_pct"`
	WorstSavingsPct   float64 `json:"worst_savings_pct"`
}

// runRecord combines Run and RunMetrics for storage.
type runRecord struct {
	Run
	RunMetrics
}

func initTelemetry(storeRoot string) error {
	return os.MkdirAll(filepath.Join(storeRoot, TelemetryDir), 0755)
}

func runsPath(storeRoot string) string {
	return filepath.Join(storeRoot, TelemetryDir, "runs.jsonl")
}

// RecordRun stores a run and its metrics.
func RecordRun(storeRoot string, run *Run, metrics *RunMetrics) error {
	if err := initTelemetry(storeRoot); err != nil {
		return err
	}

	if run.RunID == "" {
		run.RunID = generateRunID(run)
	}
	metrics.RunID = run.RunID

	// Compute derived metrics
	if metrics.BaselineEstTokens > 0 && metrics.DeltaTokens > 0 {
		metrics.TokensSaved = metrics.BaselineEstTokens - metrics.DeltaTokens
		if metrics.TokensSaved < 0 {
			metrics.TokensSaved = 0
		}
		metrics.SavingsPct = float64(metrics.TokensSaved) / float64(metrics.BaselineEstTokens) * 100
	}

	rec := runRecord{
		Run:        *run,
		RunMetrics: *metrics,
	}

	return graph.AppendRecord(runsPath(storeRoot), rec)
}

// GetMetrics retrieves the most recent N run metrics.
func GetMetrics(storeRoot string, limit int) ([]RunMetrics, error) {
	records, err := graph.ReadRecords[runRecord](runsPath(storeRoot))
	if err != nil {
		return nil, fmt.Errorf("reading runs: %w", err)
	}

	// Return most recent first
	sort.Slice(records, func(i, j int) bool {
		return records[i].EndedAt.After(records[j].EndedAt)
	})

	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}

	metrics := make([]RunMetrics, len(records))
	for i, r := range records {
		metrics[i] = r.RunMetrics
	}

	return metrics, nil
}

// GetRuns retrieves the most recent N runs with full details.
func GetRuns(storeRoot string, limit int) ([]Run, error) {
	records, err := graph.ReadRecords[runRecord](runsPath(storeRoot))
	if err != nil {
		return nil, fmt.Errorf("reading runs: %w", err)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].EndedAt.After(records[j].EndedAt)
	})

	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}

	runs := make([]Run, len(records))
	for i, r := range records {
		runs[i] = r.Run
	}

	return runs, nil
}

// ComputeROI computes aggregate ROI metrics from a set of run metrics.
func ComputeROI(metrics []RunMetrics) *ROISummary {
	if len(metrics) == 0 {
		return &ROISummary{}
	}

	summary := &ROISummary{
		TotalRuns:       len(metrics),
		BestSavingsPct:  -1,
		WorstSavingsPct: 101,
	}

	var totalSavingsPct, totalCacheHitRate, totalLatency float64

	for _, m := range metrics {
		summary.TotalBaseline += m.BaselineEstTokens
		summary.TotalDelta += m.DeltaTokens
		summary.TotalSaved += m.TokensSaved

		totalSavingsPct += m.SavingsPct
		totalCacheHitRate += m.CacheHitRate
		totalLatency += float64(m.LatencyMS)

		if m.SavingsPct > summary.BestSavingsPct {
			summary.BestSavingsPct = m.SavingsPct
		}
		if m.SavingsPct < summary.WorstSavingsPct {
			summary.WorstSavingsPct = m.SavingsPct
		}
	}

	n := float64(len(metrics))
	summary.AvgSavingsPct = totalSavingsPct / n
	summary.AvgCacheHitRate = totalCacheHitRate / n
	summary.AvgLatencyMS = totalLatency / n

	return summary
}

// EstimateBaseline estimates the token cost of scanning an entire repo (cold run).
// Uses file snapshots from the indexed commit.
func EstimateBaseline(storeRoot, commitSHA string) (int, error) {
	files, err := graph.ReadRecords[graph.FileSnapshot](graph.FilesPath(storeRoot, commitSHA))
	if err != nil {
		return 0, fmt.Errorf("reading files for %s: %w", commitSHA, err)
	}

	totalTokens := 0
	for _, f := range files {
		if f.IsBinary || f.IsGenerated {
			continue
		}
		// Approximate 0.25 tokens per byte
		totalTokens += int(float64(f.ByteSize) * 0.25)
	}

	return totalTokens, nil
}

// FormatMetrics produces a human-readable dashboard of recent metrics.
func FormatMetrics(metrics []RunMetrics, roi *ROISummary) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Token Optimization Metrics\n")
	fmt.Fprintf(&b, "═══════════════════════════════════════\n\n")

	if roi.TotalRuns == 0 {
		fmt.Fprintf(&b, "No runs recorded yet.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "Summary (%d runs):\n", roi.TotalRuns)
	fmt.Fprintf(&b, "  Total baseline tokens:  %d\n", roi.TotalBaseline)
	fmt.Fprintf(&b, "  Total delta tokens:     %d\n", roi.TotalDelta)
	fmt.Fprintf(&b, "  Total tokens saved:     %d\n", roi.TotalSaved)
	fmt.Fprintf(&b, "  Avg savings:            %.1f%%\n", roi.AvgSavingsPct)
	fmt.Fprintf(&b, "  Best savings:           %.1f%%\n", roi.BestSavingsPct)
	fmt.Fprintf(&b, "  Worst savings:          %.1f%%\n", roi.WorstSavingsPct)
	fmt.Fprintf(&b, "  Avg cache hit rate:     %.1f%%\n", roi.AvgCacheHitRate*100)
	fmt.Fprintf(&b, "  Avg latency:            %.0f ms\n", roi.AvgLatencyMS)

	if len(metrics) > 0 {
		fmt.Fprintf(&b, "\nRecent runs:\n")
		fmt.Fprintf(&b, "  %-12s  %8s  %8s  %8s  %6s\n", "Run ID", "Baseline", "Delta", "Saved", "Pct")
		fmt.Fprintf(&b, "  %-12s  %8s  %8s  %8s  %6s\n", "──────", "────────", "─────", "─────", "───")

		limit := len(metrics)
		if limit > 10 {
			limit = 10
		}
		for _, m := range metrics[:limit] {
			shortID := m.RunID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			fmt.Fprintf(&b, "  %-12s  %8d  %8d  %8d  %5.1f%%\n",
				shortID, m.BaselineEstTokens, m.DeltaTokens, m.TokensSaved, m.SavingsPct)
		}
	}

	return b.String()
}

func generateRunID(run *Run) string {
	data := fmt.Sprintf("%s:%s:%s:%d", run.Repo, run.HeadCommit, run.TaskHash, run.StartedAt.UnixNano())
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:8])
}
