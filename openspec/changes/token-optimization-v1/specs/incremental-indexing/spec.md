## ADDED Requirements

### Requirement: Index a single commit
The system SHALL provide an `IndexCommit` function that creates file snapshots, path records, and a commit record for a given commit SHA.

#### Scenario: Index a commit with multiple files
- **WHEN** `IndexCommit` is called for a commit containing N files
- **THEN** the system creates N file snapshot records in `.ctx/graph/snapshots/<SHA>/files.jsonl`, N path records in `.ctx/graph/manifests/paths.jsonl`, and 1 commit record in `.ctx/graph/manifests/commits.jsonl`

#### Scenario: Index is idempotent
- **WHEN** `IndexCommit` is called twice for the same commit
- **THEN** the second call is a no-op (files.jsonl already exists, no duplicate records)

#### Scenario: Incremental path tracking
- **WHEN** a file exists in commit A and commit B
- **THEN** the path record is created during indexing of commit A and reused (not duplicated) during indexing of commit B

### Requirement: Index a range of commits
The system SHALL provide an `IndexRange` function that indexes all commits between a base (exclusive) and head (inclusive) commit.

#### Scenario: Index a range of 3 commits
- **WHEN** `IndexRange` is called with base=commit1 and head=commit3
- **THEN** commit2 and commit3 are indexed (commit1 is excluded as the base)

### Requirement: File snapshot metadata
Each file snapshot SHALL include: commit SHA, path ID, git blob OID, content SHA-256 hash, detected language, byte size, line count (LOC), and flags for binary and generated files.

#### Scenario: Go file snapshot
- **WHEN** a `.go` file is indexed
- **THEN** the snapshot has `language: "go"`, accurate LOC count, and `is_binary: false`

#### Scenario: Binary file snapshot
- **WHEN** a binary file (containing null bytes) is indexed
- **THEN** the snapshot has `is_binary: true` and `loc: 0`

#### Scenario: Generated file detection
- **WHEN** a file at path `vendor/lib.go` or `node_modules/pkg/index.js` is indexed
- **THEN** the snapshot has `is_generated: true`

### Requirement: Language detection
The system SHALL detect programming language from file extensions for at least: Go, TypeScript, JavaScript, Python, Rust, Java, Ruby, C, C++, C#, Swift, Kotlin, Markdown, YAML, JSON, TOML, XML, HTML, CSS, SCSS, SQL, Shell, Dockerfile, and Protobuf.

#### Scenario: Detect Go language
- **WHEN** a file with extension `.go` is indexed
- **THEN** the detected language is `"go"`

#### Scenario: Detect TypeScript language
- **WHEN** a file with extension `.ts` or `.tsx` is indexed
- **THEN** the detected language is `"typescript"`

#### Scenario: Filename-based detection
- **WHEN** a file named `Dockerfile` (no extension) is indexed
- **THEN** the detected language is `"dockerfile"`

### Requirement: CLI index command
The system SHALL provide a `ctx index` command that indexes HEAD or a specified commit.

#### Scenario: Index HEAD
- **WHEN** the user runs `ctx index` without flags
- **THEN** the system indexes the current HEAD commit

#### Scenario: Index specific commit
- **WHEN** the user runs `ctx index --commit <SHA>`
- **THEN** the system indexes the specified commit

#### Scenario: Index requires git repository
- **WHEN** the user runs `ctx index` outside a git repository
- **THEN** the system reports an error

### Requirement: Delta computation
The system SHALL provide a `ComputeDelta` function that compares file snapshots of two indexed commits and reports changed, added, and deleted files.

#### Scenario: Compute delta with changes
- **WHEN** commit B modifies file A, adds file C, and deletes file D compared to commit A
- **THEN** `ComputeDelta(A, B)` reports file A as changed, file C as added, and file D as deleted

#### Scenario: Self-delta is empty
- **WHEN** `ComputeDelta` is called with the same commit for base and head
- **THEN** the report has no changes

#### Scenario: Delta output formats
- **WHEN** a DeltaReport is generated
- **THEN** it can be serialized to JSON via `.JSON()` and to human-readable text via `.Human()`

### Requirement: CLI delta command
The system SHALL provide a `ctx delta` command that compares two indexed commits.

#### Scenario: Delta with JSON output
- **WHEN** the user runs `ctx delta --base X --head Y`
- **THEN** the system outputs a JSON delta report

#### Scenario: Delta with human output
- **WHEN** the user runs `ctx delta --base X --head Y --human`
- **THEN** the system outputs a human-readable delta summary

#### Scenario: Delta requires both flags
- **WHEN** the user runs `ctx delta` without `--base` or `--head`
- **THEN** the system reports that both flags are required
