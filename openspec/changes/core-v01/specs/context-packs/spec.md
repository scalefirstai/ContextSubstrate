## ADDED Requirements

### Requirement: Initialize context store
The system SHALL provide a `ctx init` command that creates a `.ctx/` directory in the current working directory with the required subdirectory structure (`objects/`, `packs/`, `refs/`, `config.json`).

#### Scenario: Initialize in a new directory
- **WHEN** the user runs `ctx init` in a directory without a `.ctx/` directory
- **THEN** the system creates `.ctx/` with `objects/`, `packs/`, `refs/` subdirectories and a `config.json` file

#### Scenario: Initialize in a directory that already has a store
- **WHEN** the user runs `ctx init` in a directory that already contains a `.ctx/` directory
- **THEN** the system reports that the store already exists and makes no changes

### Requirement: Create a context pack from execution log
The system SHALL provide a `ctx pack` command that accepts an execution log (JSON) and produces an immutable, content-addressed Context Pack. The pack manifest SHALL be a JSON file referencing content-addressed blobs for all large content.

#### Scenario: Pack a valid execution log
- **WHEN** the user runs `ctx pack <log-file>` with a valid JSON execution log
- **THEN** the system stores all content as blobs in `.ctx/objects/`, creates a pack manifest, and outputs the pack hash in `ctx://<sha256>` format

#### Scenario: Pack with duplicate content
- **WHEN** the user packs a log that contains input files identical to previously stored blobs
- **THEN** the system reuses existing blobs (no duplicate storage) and the pack manifest references the existing object hashes

#### Scenario: Pack an invalid execution log
- **WHEN** the user runs `ctx pack` with a malformed or incomplete JSON log
- **THEN** the system exits with a non-zero status and reports which required fields are missing or invalid

### Requirement: Context pack manifest format
The pack manifest SHALL be a JSON document containing: `version`, `hash`, `created` timestamp, `model` (identifier and parameters), `system_prompt` reference, `prompts` array, `inputs` array, `steps` array (with index, type, tool, parameters, output reference, deterministic flag, timestamp), `outputs` array, and `environment` metadata (OS, runtime, tool versions).

#### Scenario: Manifest contains all required fields
- **WHEN** a context pack is created from a complete execution log
- **THEN** the resulting manifest JSON contains all required fields: `version`, `hash`, `created`, `model`, `system_prompt`, `prompts`, `inputs`, `steps`, `outputs`, and `environment`

#### Scenario: Manifest hash is deterministic
- **WHEN** the same execution log is packed twice
- **THEN** both packs produce identical manifest hashes (canonical JSON serialization with sorted keys)

### Requirement: Content-addressed blob storage
All content referenced by a pack manifest SHALL be stored as content-addressed blobs in `.ctx/objects/` using SHA-256 hashing. Blobs SHALL be stored in a two-character prefix subdirectory structure (e.g., `objects/ab/cdef1234...`).

#### Scenario: Blob stored with correct path
- **WHEN** content with SHA-256 hash `abcdef1234...` is stored
- **THEN** the blob file is written to `.ctx/objects/ab/cdef1234...`

#### Scenario: Blob content integrity
- **WHEN** a blob is read from the object store
- **THEN** recomputing SHA-256 over the blob content matches the filename

### Requirement: Inspect a context pack
The system SHALL provide a `ctx show <hash>` command that displays the contents of a context pack in human-readable format.

#### Scenario: Show an existing pack
- **WHEN** the user runs `ctx show <hash>` with a valid pack hash
- **THEN** the system displays the pack manifest with resolved blob content summaries (file names, sizes, tool calls)

#### Scenario: Show a non-existent pack
- **WHEN** the user runs `ctx show <hash>` with a hash that does not exist in the store
- **THEN** the system exits with a non-zero status and reports the pack was not found

### Requirement: Context pack immutability
Once a context pack is finalized and stored, its contents SHALL NOT be modifiable. Any attempt to alter a stored pack SHALL be rejected.

#### Scenario: Attempt to modify a finalized pack
- **WHEN** a process attempts to overwrite a blob or manifest that already exists in the store
- **THEN** the system rejects the write and reports that the object is immutable

### Requirement: Context pack portability
Context packs SHALL be self-contained and portable. All content referenced by the manifest SHALL be resolvable from the local `.ctx/objects/` store without external dependencies.

#### Scenario: Pack transferred to another machine
- **WHEN** the `.ctx/` directory is copied to another machine with the same OS
- **THEN** all pack manifests and their referenced blobs are intact and readable by `ctx show`
