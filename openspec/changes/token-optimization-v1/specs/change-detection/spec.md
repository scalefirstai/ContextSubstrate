## ADDED Requirements

### Requirement: Detect file changes between commits
The system SHALL provide a `DetectChanges` function that uses `git diff-tree` to identify files added, modified, and deleted between two commits.

#### Scenario: Detect added files
- **WHEN** a commit adds a new file
- **THEN** `DetectChanges` includes that file in the `FilesAdded` list

#### Scenario: Detect modified files
- **WHEN** a commit modifies an existing file
- **THEN** `DetectChanges` includes that file in the `FilesChanged` list

#### Scenario: Detect deleted files
- **WHEN** a commit removes a file
- **THEN** `DetectChanges` includes that file in the `FilesDeleted` list

#### Scenario: Detect renamed files
- **WHEN** a commit renames a file
- **THEN** `DetectChanges` records the old path as deleted and the new path as added

#### Scenario: No changes between identical commits
- **WHEN** `DetectChanges` is called with the same SHA for base and head
- **THEN** all change lists are empty

### Requirement: List files at a commit
The system SHALL provide a `ListFilesAtCommit` function that returns all tracked files at a given commit using `git ls-tree`.

#### Scenario: List files at first commit
- **WHEN** `ListFilesAtCommit` is called with the first commit SHA
- **THEN** it returns all files that were present in that commit

### Requirement: Get commit metadata
The system SHALL provide a `GetCommitInfo` function that retrieves SHA, parent SHA, author, timestamp, and commit message.

#### Scenario: Get commit info for a valid commit
- **WHEN** `GetCommitInfo` is called with a valid commit SHA
- **THEN** it returns the correct SHA, parent, author, timestamp, and message

#### Scenario: Get commit info for merge commit
- **WHEN** `GetCommitInfo` is called for a merge commit with multiple parents
- **THEN** it returns the first parent SHA

### Requirement: Get HEAD SHA
The system SHALL provide a `GetHeadSHA` function that returns the full SHA of the current HEAD.

#### Scenario: Get HEAD in a valid repository
- **WHEN** `GetHeadSHA` is called in a git repository
- **THEN** it returns the 40-character SHA of HEAD

### Requirement: Get repository root
The system SHALL provide a `GetRepoRoot` function that returns the root directory of the git repository.

#### Scenario: Get repo root from subdirectory
- **WHEN** `GetRepoRoot` is called from a subdirectory of a git repository
- **THEN** it returns the repository root directory

#### Scenario: Get repo root from non-repository
- **WHEN** `GetRepoRoot` is called from a directory that is not part of a git repository
- **THEN** it returns an error indicating no git repository was found
