## ADDED Requirements

### Requirement: Record agent run metrics
The system SHALL provide a RecordRun function that stores run metadata and token metrics in `.ctx/telemetry/runs.jsonl`.

#### Scenario: Record a completed run
- **WHEN** RecordRun is called with a Run and RunMetrics
- **THEN** the run is stored with auto-generated RunID, and derived metrics (TokensSaved, SavingsPct) are computed

#### Scenario: Auto-compute savings
- **WHEN** RecordRun is called with BaselineEstTokens=10000 and DeltaTokens=2000
- **THEN** TokensSaved is set to 8000 and SavingsPct is set to 80.0

### Requirement: Retrieve recent metrics
The system SHALL provide a GetMetrics function that returns the N most recent run metrics, ordered by end time descending.

#### Scenario: Get metrics with limit
- **WHEN** GetMetrics is called with limit=3 and 5 runs exist
- **THEN** the 3 most recent runs are returned

#### Scenario: Get metrics from empty store
- **WHEN** GetMetrics is called on a store with no recorded runs
- **THEN** an empty slice is returned with no error

### Requirement: ROI computation
The system SHALL provide a ComputeROI function that aggregates metrics across runs into a summary including totals, averages, and best/worst savings.

#### Scenario: Compute ROI from multiple runs
- **WHEN** ComputeROI is called with metrics from 3 runs
- **THEN** TotalRuns, TotalBaseline, TotalDelta, TotalSaved, AvgSavingsPct, BestSavingsPct, and WorstSavingsPct are computed

#### Scenario: Compute ROI from empty input
- **WHEN** ComputeROI is called with nil or empty metrics
- **THEN** a zero-valued ROISummary is returned

### Requirement: Baseline estimation
The system SHALL provide an EstimateBaseline function that estimates total token cost of scanning a full repo at a given commit.

#### Scenario: Estimate baseline tokens
- **WHEN** EstimateBaseline is called with an indexed commit
- **THEN** it returns the sum of estimated tokens for all non-binary, non-generated files (0.25 tokens/byte)

### Requirement: Metrics dashboard
The system SHALL provide a FormatMetrics function that produces a human-readable dashboard of token savings.

#### Scenario: Format metrics with data
- **WHEN** FormatMetrics is called with metrics and ROI summary
- **THEN** a formatted dashboard is returned showing totals, averages, and per-run details

#### Scenario: Format metrics empty
- **WHEN** FormatMetrics is called with no data
- **THEN** "No runs recorded yet" is displayed

### Requirement: CLI metrics command
The system SHALL provide a `ctx metrics` command that displays the token savings dashboard.

#### Scenario: Display metrics
- **WHEN** the user runs `ctx metrics`
- **THEN** the system displays recent run metrics and ROI summary

#### Scenario: Metrics with limit
- **WHEN** the user runs `ctx metrics --limit 5`
- **THEN** at most 5 recent runs are displayed
