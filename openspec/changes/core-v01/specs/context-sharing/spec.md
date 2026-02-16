## ADDED Requirements

### Requirement: Hash-based pack references
Context packs SHALL be referenced exclusively by their SHA-256 content hash. The hash serves as the universal identifier across all operations (show, replay, diff, verify, fork).

#### Scenario: Reference a pack by hash
- **WHEN** a user provides a pack hash to any `ctx` command
- **THEN** the system resolves the pack from `.ctx/objects/` using the hash without requiring any other identifier

#### Scenario: Hash uniquely identifies content
- **WHEN** two packs have different content
- **THEN** they have different SHA-256 hashes

### Requirement: Fork a context pack
The system SHALL provide a `ctx fork <hash>` command that creates a new mutable draft derived from an existing pack. The draft preserves the parent pack's content and records the parent hash as lineage.

#### Scenario: Fork an existing pack
- **WHEN** the user runs `ctx fork <hash>` with a valid pack hash
- **THEN** the system creates a new draft pack with the parent's content, adds a `parent` field referencing the original hash, and outputs the draft location

#### Scenario: Fork a non-existent pack
- **WHEN** the user runs `ctx fork <hash>` with a hash not found in the store
- **THEN** the system exits with non-zero status and reports the pack was not found

#### Scenario: Forked pack records lineage
- **WHEN** a forked draft is finalized into a new pack
- **THEN** the new pack manifest contains a `parent` field with the original pack's hash

### Requirement: List context packs
The system SHALL provide a `ctx log` command that lists all finalized context packs in the store, ordered by creation date (newest first).

#### Scenario: List packs in a populated store
- **WHEN** the user runs `ctx log` in a directory with stored packs
- **THEN** the system displays each pack's hash, creation date, model identifier, and step count

#### Scenario: List packs in an empty store
- **WHEN** the user runs `ctx log` in a directory with no stored packs
- **THEN** the system reports that no packs were found

### Requirement: Pack shareability without environment coupling
Sharing a context pack SHALL require only copying the relevant objects from `.ctx/objects/`. No environment-specific paths, credentials, or configuration SHALL be embedded in pack manifests.

#### Scenario: Pack shared between users
- **WHEN** user A copies the objects referenced by a pack hash to user B's `.ctx/objects/`
- **THEN** user B can run `ctx show`, `ctx diff`, and `ctx replay` on the pack without additional setup

#### Scenario: No absolute paths in manifest
- **WHEN** a pack manifest is inspected
- **THEN** no field contains absolute filesystem paths — all file references use content hashes or relative names

### Requirement: Pack comparability via diff
Any two context packs referenced by hash SHALL be comparable using `ctx diff`. The sharing mechanism (hash-based references) SHALL be compatible with the drift detection capability.

#### Scenario: Diff packs from different users
- **WHEN** user A and user B each have a pack and both packs exist in the local store
- **THEN** `ctx diff <hash-a> <hash-b>` produces a valid drift report regardless of which user created each pack
