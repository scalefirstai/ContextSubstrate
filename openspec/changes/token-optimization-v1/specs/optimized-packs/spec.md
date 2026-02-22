## ADDED Requirements

### Requirement: Token-budgeted pack generation
The system SHALL provide a GeneratePack function that selects relevant files and symbols within a token budget for a given task description.

#### Scenario: Generate pack for a task
- **WHEN** GeneratePack is called with a task description and indexed commit
- **THEN** an OptimizedPack is produced containing ranked files and symbols that fit within the token budget

#### Scenario: Default token cap
- **WHEN** GeneratePack is called with TokenCap of 0
- **THEN** the default token cap of 32000 is used

### Requirement: Task-based relevance scoring
The system SHALL score files and symbols based on relevance to the task description using keyword matching.

#### Scenario: Task keywords boost file ranking
- **WHEN** a task description contains "authentication" and the repo has an auth.go file
- **THEN** auth.go is ranked higher than unrelated files

#### Scenario: Exported symbols score higher
- **WHEN** an exported function matches task keywords
- **THEN** it receives a higher relevance score than a private function

### Requirement: File filtering
The system SHALL exclude binary files, generated files, and (optionally) test files from optimized packs.

#### Scenario: Binary files excluded
- **WHEN** the indexed commit contains binary files
- **THEN** binary files are not included in the optimized pack

#### Scenario: Test files excluded by default
- **WHEN** GeneratePack is called without IncludeTests
- **THEN** files matching test patterns (_test.go, .test.ts, .spec.js, etc.) are excluded

#### Scenario: Test files included when requested
- **WHEN** GeneratePack is called with IncludeTests=true
- **THEN** test files are included in the pack

### Requirement: Token estimation
The system SHALL estimate token counts for files and symbols based on byte size, using an approximate ratio of 0.25 tokens per byte.

### Requirement: Pack output formats
The system SHALL support JSON and human-readable output formats for OptimizedPack.

#### Scenario: JSON output
- **WHEN** OptimizedPack.JSON() is called
- **THEN** valid formatted JSON is returned containing commit, task, files, symbols, and token estimates

#### Scenario: Human output
- **WHEN** OptimizedPack.Human() is called
- **THEN** a readable summary is returned showing token budget usage, file list with token estimates, and symbol list

### Requirement: CLI optimize command
The system SHALL provide a `ctx optimize` command with required `--task` flag and optional `--commit`, `--token-cap`, `--include-tests`, and `--human` flags.

#### Scenario: Optimize with task flag
- **WHEN** the user runs `ctx optimize --task "add auth"`
- **THEN** the system generates and outputs an optimized pack for HEAD

#### Scenario: Optimize without task flag
- **WHEN** the user runs `ctx optimize` without --task
- **THEN** the system reports that --task is required
