## ADDED Requirements

### Requirement: Regex-based symbol extraction
The system SHALL extract symbol definitions from source files using regex patterns for Go, TypeScript/JavaScript, and Python.

#### Scenario: Extract Go symbols
- **WHEN** a Go file containing functions, methods, types, interfaces, constants, and variables is indexed
- **THEN** the system produces SymbolRecord entries for each definition with correct kind, name, visibility, and signature

#### Scenario: Extract TypeScript symbols
- **WHEN** a TypeScript file containing exported functions, arrow functions, classes, interfaces, and type aliases is indexed
- **THEN** the system produces SymbolRecord entries for each with correct export visibility detection

#### Scenario: Extract Python symbols
- **WHEN** a Python file containing classes, functions, and methods is indexed
- **THEN** the system produces SymbolRecord entries with correct kind detection and underscore-based visibility

#### Scenario: Unsupported language returns empty
- **WHEN** a file with an unsupported language is passed to ExtractSymbols
- **THEN** nil is returned for both symbols and regions

### Requirement: Region records for symbol definitions
The system SHALL create RegionRecord entries for each extracted symbol, capturing the start and end line of the definition.

#### Scenario: Regions correspond to symbols
- **WHEN** symbols are extracted from a source file
- **THEN** each symbol has a corresponding RegionRecord with matching DefRegionID

### Requirement: Go visibility detection
The system SHALL detect Go symbol visibility based on capitalization: uppercase initial = exported, lowercase = private.

### Requirement: Import edge extraction
The system SHALL extract import/dependency edges from source files.

#### Scenario: Go imports
- **WHEN** a Go file with single and block import statements is indexed
- **THEN** the system creates ImportEdge records for each imported package

#### Scenario: TypeScript imports
- **WHEN** a TypeScript file with import/from, require(), and dynamic import() is indexed
- **THEN** the system creates ImportEdge records for each imported module

#### Scenario: Python imports
- **WHEN** a Python file with import and from/import statements is indexed
- **THEN** the system creates ImportEdge records for each imported module

### Requirement: Call edge extraction
The system SHALL detect basic function call patterns within symbol regions using grep-based matching.

#### Scenario: Internal call resolution
- **WHEN** a function calls another function defined in the same commit
- **THEN** a CallEdge is created with ToSymbolID set to the target's ID and higher confidence (0.8)

#### Scenario: External call detection
- **WHEN** a function calls a name not found in known symbols
- **THEN** a CallEdge is created with ToExternalRef and lower confidence (0.5)

#### Scenario: Keyword filtering
- **WHEN** call detection encounters language keywords (if, for, return, etc.)
- **THEN** the keywords are not recorded as call edges

### Requirement: Integration with IndexCommit
The system SHALL automatically extract symbols, regions, import edges, and call edges during IndexCommit for supported languages.

#### Scenario: IndexCommit produces symbol files
- **WHEN** IndexCommit indexes a commit containing Go source files
- **THEN** symbols.jsonl, regions.jsonl, edges.imports.jsonl, and edges.calls.jsonl are created under the commit's snapshot directory
