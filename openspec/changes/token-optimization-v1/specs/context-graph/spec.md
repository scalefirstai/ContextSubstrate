## ADDED Requirements

### Requirement: JSONL graph store initialization
The system SHALL create a `.ctx/graph/` directory structure during `ctx init` containing `manifests/` and `snapshots/` subdirectories.

#### Scenario: Initialize store creates graph directories
- **WHEN** the user runs `ctx init` in a new directory
- **THEN** the system creates `.ctx/graph/manifests/` and `.ctx/graph/snapshots/` in addition to the existing store structure

#### Scenario: Graph initialization is idempotent
- **WHEN** `graph.InitGraph` is called on a store that already has graph directories
- **THEN** no error occurs and existing data is preserved

### Requirement: JSONL record types
The system SHALL define typed Go structs for all graph records: CommitRecord, PathRecord, FileSnapshot, SymbolRecord, RegionRecord, ImportEdge, and CallEdge. Each record SHALL have a `type` field identifying its kind.

#### Scenario: Record types are JSON-serializable
- **WHEN** any graph record is marshaled to JSON
- **THEN** the output is a valid single-line JSON object with all fields properly encoded

### Requirement: JSONL read/write utilities
The system SHALL provide `AppendRecord`, `ReadRecords`, and `WriteRecords` functions for JSONL file I/O.

#### Scenario: Append record to new file
- **WHEN** `AppendRecord` is called with a path that does not exist
- **THEN** the file and parent directories are created, and the record is written as a single JSON line

#### Scenario: Append record to existing file
- **WHEN** `AppendRecord` is called with a path to an existing JSONL file
- **THEN** the new record is appended as a new line without modifying existing content

#### Scenario: Read records from nonexistent file
- **WHEN** `ReadRecords` is called with a path that does not exist
- **THEN** an empty slice is returned with no error

#### Scenario: Read records round-trip
- **WHEN** records are written with `AppendRecord` or `WriteRecords` and read back with `ReadRecords`
- **THEN** all records are deserialized with identical field values

#### Scenario: WriteRecords replaces file content
- **WHEN** `WriteRecords` is called on an existing JSONL file
- **THEN** the file content is completely replaced with the new records

### Requirement: Deterministic JSONL output
File snapshots SHALL be written in sorted order (by PathID) to ensure deterministic, reproducible JSONL files.

#### Scenario: File snapshots are sorted
- **WHEN** `IndexCommit` writes file snapshots for a commit
- **THEN** the records in `files.jsonl` are sorted by PathID

### Requirement: Path helper functions
The system SHALL provide functions to compute standard paths within the graph store: CommitsPath, PathsPath, FilesPath, SnapshotDir, SymbolsPath, RegionsPath, ImportEdgesPath, CallEdgesPath.

#### Scenario: Path functions return correct locations
- **WHEN** any path helper is called with a store root and commit SHA
- **THEN** the returned path follows the layout `.ctx/graph/{manifests|snapshots}/<subpath>`
