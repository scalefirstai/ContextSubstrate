package telemetry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestStore(t *testing.T) string {
	t.Helper()
	storeRoot := filepath.Join(t.TempDir(), ".ctx")
	os.MkdirAll(storeRoot, 0755)
	return storeRoot
}

func TestRecordAndGetMetrics(t *testing.T) {
	storeRoot := setupTestStore(t)

	run := &Run{
		Repo:       "/tmp/repo",
		BaseCommit: "aaa",
		HeadCommit: "bbb",
		Agent:      "claude",
		TaskHash:   "task1",
		Status:     "completed",
		StartedAt:  time.Now().Add(-5 * time.Second),
		EndedAt:    time.Now(),
	}

	metrics := &RunMetrics{
		BaselineEstTokens:  10000,
		DeltaTokens:        2000,
		CacheHitRate:       0.75,
		FilesInvalidated:   3,
		SymbolsInvalidated: 10,
		LatencyMS:          500,
	}

	if err := RecordRun(storeRoot, run, metrics); err != nil {
		t.Fatalf("RecordRun: %v", err)
	}

	// Verify derived metrics were computed
	if metrics.TokensSaved != 8000 {
		t.Errorf("TokensSaved: got %d, want 8000", metrics.TokensSaved)
	}
	if metrics.SavingsPct < 79 || metrics.SavingsPct > 81 {
		t.Errorf("SavingsPct: got %.1f, want ~80.0", metrics.SavingsPct)
	}

	// Retrieve
	got, err := GetMetrics(storeRoot, 10)
	if err != nil {
		t.Fatalf("GetMetrics: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(got))
	}
	if got[0].BaselineEstTokens != 10000 {
		t.Errorf("BaselineEstTokens: got %d, want 10000", got[0].BaselineEstTokens)
	}
	if got[0].TokensSaved != 8000 {
		t.Errorf("TokensSaved: got %d, want 8000", got[0].TokensSaved)
	}
}

func TestGetMetricsEmpty(t *testing.T) {
	storeRoot := setupTestStore(t)

	metrics, err := GetMetrics(storeRoot, 10)
	if err != nil {
		t.Fatalf("GetMetrics: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(metrics))
	}
}

func TestGetMetricsLimit(t *testing.T) {
	storeRoot := setupTestStore(t)

	// Record 5 runs
	for i := 0; i < 5; i++ {
		run := &Run{
			Repo:       "/repo",
			HeadCommit: "abc",
			Status:     "completed",
			StartedAt:  time.Now().Add(time.Duration(-5+i) * time.Second),
			EndedAt:    time.Now().Add(time.Duration(i) * time.Second),
		}
		metrics := &RunMetrics{
			BaselineEstTokens: 1000 * (i + 1),
			DeltaTokens:       500,
		}
		if err := RecordRun(storeRoot, run, metrics); err != nil {
			t.Fatalf("RecordRun %d: %v", i, err)
		}
	}

	// Limit to 3
	metrics, err := GetMetrics(storeRoot, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 3 {
		t.Fatalf("expected 3 metrics, got %d", len(metrics))
	}

	// Should be most recent first
	if metrics[0].BaselineEstTokens < metrics[2].BaselineEstTokens {
		t.Error("expected most recent runs first (higher baseline)")
	}
}

func TestGetRuns(t *testing.T) {
	storeRoot := setupTestStore(t)

	run := &Run{
		Repo:       "/repo",
		HeadCommit: "abc",
		Agent:      "agent1",
		Status:     "completed",
		StartedAt:  time.Now(),
		EndedAt:    time.Now(),
	}
	RecordRun(storeRoot, run, &RunMetrics{BaselineEstTokens: 100, DeltaTokens: 50})

	runs, err := GetRuns(storeRoot, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].Agent != "agent1" {
		t.Errorf("Agent: got %q, want %q", runs[0].Agent, "agent1")
	}
}

func TestComputeROI(t *testing.T) {
	metrics := []RunMetrics{
		{BaselineEstTokens: 10000, DeltaTokens: 2000, TokensSaved: 8000, SavingsPct: 80.0, CacheHitRate: 0.9, LatencyMS: 100},
		{BaselineEstTokens: 8000, DeltaTokens: 4000, TokensSaved: 4000, SavingsPct: 50.0, CacheHitRate: 0.6, LatencyMS: 200},
		{BaselineEstTokens: 12000, DeltaTokens: 1000, TokensSaved: 11000, SavingsPct: 91.7, CacheHitRate: 0.95, LatencyMS: 50},
	}

	roi := ComputeROI(metrics)

	if roi.TotalRuns != 3 {
		t.Errorf("TotalRuns: got %d, want 3", roi.TotalRuns)
	}
	if roi.TotalBaseline != 30000 {
		t.Errorf("TotalBaseline: got %d, want 30000", roi.TotalBaseline)
	}
	if roi.TotalDelta != 7000 {
		t.Errorf("TotalDelta: got %d, want 7000", roi.TotalDelta)
	}
	if roi.TotalSaved != 23000 {
		t.Errorf("TotalSaved: got %d, want 23000", roi.TotalSaved)
	}
	if roi.BestSavingsPct < 91.0 {
		t.Errorf("BestSavingsPct: got %.1f, want ~91.7", roi.BestSavingsPct)
	}
	if roi.WorstSavingsPct > 51.0 {
		t.Errorf("WorstSavingsPct: got %.1f, want ~50.0", roi.WorstSavingsPct)
	}
}

func TestComputeROIEmpty(t *testing.T) {
	roi := ComputeROI(nil)
	if roi.TotalRuns != 0 {
		t.Errorf("expected 0 total runs for empty input")
	}
}

func TestFormatMetrics(t *testing.T) {
	metrics := []RunMetrics{
		{RunID: "run1234567890", BaselineEstTokens: 10000, DeltaTokens: 2000, TokensSaved: 8000, SavingsPct: 80.0},
	}
	roi := ComputeROI(metrics)

	output := FormatMetrics(metrics, roi)
	if output == "" {
		t.Fatal("FormatMetrics returned empty string")
	}

	// Check key sections exist
	for _, want := range []string{"Token Optimization", "Summary", "Total baseline", "run123456789"} {
		if !containsStr(output, want) {
			t.Errorf("missing %q in output:\n%s", want, output)
		}
	}
}

func TestFormatMetricsEmpty(t *testing.T) {
	roi := ComputeROI(nil)
	output := FormatMetrics(nil, roi)
	if !containsStr(output, "No runs recorded") {
		t.Errorf("expected 'No runs recorded' in: %s", output)
	}
}

func TestRunIDGeneration(t *testing.T) {
	storeRoot := setupTestStore(t)

	run := &Run{
		Repo:      "/repo",
		Status:    "completed",
		StartedAt: time.Now(),
		EndedAt:   time.Now(),
	}

	RecordRun(storeRoot, run, &RunMetrics{})

	if run.RunID == "" {
		t.Error("RunID should be auto-generated")
	}
	if len(run.RunID) != 16 { // 8 bytes hex-encoded
		t.Errorf("RunID length: got %d, want 16", len(run.RunID))
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
