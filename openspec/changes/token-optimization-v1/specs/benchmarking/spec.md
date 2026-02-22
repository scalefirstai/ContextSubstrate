## ADDED Requirements

### Requirement: Cold vs warm benchmarking
The system SHALL provide a `ctx benchmark` command that compares cold (full repo scan) vs warm (incremental delta) token usage across recent commits.

#### Scenario: Benchmark across commits
- **WHEN** the user runs `ctx benchmark --commits 10`
- **THEN** the system indexes the 10 most recent commits, computes cold vs warm estimates for each pair, and displays a comparison table

#### Scenario: Benchmark with insufficient commits
- **WHEN** the user runs `ctx benchmark` in a repo with fewer than 2 commits
- **THEN** the system reports that at least 2 commits are needed

### Requirement: Benchmark output format
The benchmark output SHALL display a table with columns: Commit (short SHA), Cold (estimated baseline tokens), Warm (estimated delta tokens), Saved (difference), and Pct (savings percentage).

#### Scenario: Benchmark table format
- **WHEN** the benchmark completes
- **THEN** each row shows the head commit's metrics compared to its preceding commit

### Requirement: Automatic indexing during benchmark
The system SHALL automatically index all commits in the benchmark range before computing estimates.

#### Scenario: Benchmark indexes unindexed commits
- **WHEN** benchmark is run and some commits are not yet indexed
- **THEN** those commits are indexed as part of the benchmark process
