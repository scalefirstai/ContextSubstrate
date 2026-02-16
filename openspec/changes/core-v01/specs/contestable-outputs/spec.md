## ADDED Requirements

### Requirement: Artifact provenance metadata
Every artifact produced by an agent run SHALL have an associated sidecar metadata file (`<artifact>.ctx.json`) containing provenance information: the producing context pack hash, supporting input references, tools involved, and optional confidence/uncertainty notes.

#### Scenario: Sidecar file created with artifact
- **WHEN** a context pack produces an output artifact `result.md`
- **THEN** a sidecar file `result.md.ctx.json` is created alongside it containing `context_pack`, `inputs`, `tools`, and optional `confidence` and `notes` fields

#### Scenario: Sidecar contains valid pack reference
- **WHEN** a sidecar file is read
- **THEN** the `context_pack` field contains a valid SHA-256 hash that resolves to an existing pack in the store

### Requirement: Verify artifact provenance
The system SHALL provide a `ctx verify <artifact>` command that checks an artifact's provenance by validating its sidecar metadata against the context store.

#### Scenario: Verify an artifact with valid provenance
- **WHEN** the user runs `ctx verify result.md` and `result.md.ctx.json` exists with a valid pack reference
- **THEN** the system confirms the artifact is traceable, showing the pack hash, creation date, and tools involved

#### Scenario: Verify an artifact with no sidecar
- **WHEN** the user runs `ctx verify result.md` and no `result.md.ctx.json` file exists
- **THEN** the system reports the artifact has no provenance metadata and exits with non-zero status

#### Scenario: Verify an artifact with broken provenance
- **WHEN** the user runs `ctx verify result.md` and the sidecar references a pack hash not found in the store
- **THEN** the system reports the provenance is broken (pack not found) and exits with non-zero status

### Requirement: Artifact content integrity check
The `ctx verify` command SHALL hash the artifact content and check whether it matches the output recorded in the referenced context pack.

#### Scenario: Artifact content matches pack record
- **WHEN** the artifact's SHA-256 hash matches the `content_ref` in the referenced pack's outputs
- **THEN** the verification reports content integrity as "verified"

#### Scenario: Artifact content has been modified
- **WHEN** the artifact's SHA-256 hash does not match the `content_ref` in the referenced pack's outputs
- **THEN** the verification reports content integrity as "modified" with the expected and actual hashes

### Requirement: Confidence and uncertainty notes
The sidecar metadata format SHALL support optional `confidence` (string: "high", "medium", "low") and `notes` (free-text string) fields for expressing uncertainty about an artifact.

#### Scenario: Artifact with confidence metadata
- **WHEN** an artifact's sidecar includes `"confidence": "low"` and `"notes": "Multiple valid approaches exist"`
- **THEN** `ctx verify` displays the confidence level and notes alongside the provenance information

#### Scenario: Artifact without confidence metadata
- **WHEN** an artifact's sidecar omits `confidence` and `notes` fields
- **THEN** `ctx verify` reports provenance without confidence information (no error)

### Requirement: Contestability guarantee
Any verified artifact SHALL provide sufficient information for a human to replay the producing context pack and review the decisions that led to the artifact.

#### Scenario: Challenge an artifact
- **WHEN** a user runs `ctx verify result.md` and the provenance is valid
- **THEN** the output includes the pack hash suitable for use with `ctx replay` and `ctx show`, enabling the user to inspect or re-execute the producing run
