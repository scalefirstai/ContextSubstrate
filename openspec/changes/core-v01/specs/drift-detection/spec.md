## ADDED Requirements

### Requirement: Diff two context packs
The system SHALL provide a `ctx diff <hash-a> <hash-b>` command that compares two context packs and produces a structured drift report.

#### Scenario: Diff two existing packs
- **WHEN** the user runs `ctx diff <hash-a> <hash-b>` with two valid pack hashes
- **THEN** the system outputs a drift report comparing the two packs section by section

#### Scenario: Diff with a non-existent pack
- **WHEN** the user runs `ctx diff` and one or both hashes are not found in the store
- **THEN** the system exits with non-zero status and reports which pack was not found

#### Scenario: Diff identical packs
- **WHEN** the user runs `ctx diff` with the same hash for both arguments
- **THEN** the system reports no drift detected

### Requirement: Typed drift categories
The diff report SHALL categorize drift into typed entries: `prompt_drift` (system prompt or user prompts differ), `tool_drift` (different tools called or different call order), `param_drift` (same tool with different parameters), `reasoning_drift` (intermediate step outputs diverge), and `output_drift` (final outputs differ).

#### Scenario: Prompt changed between packs
- **WHEN** pack A and pack B have different system prompts or user prompt content
- **THEN** the drift report contains a `prompt_drift` entry identifying which prompts differ

#### Scenario: Different tools called
- **WHEN** pack A calls tools [read_file, write_file] and pack B calls [read_file, execute_command]
- **THEN** the drift report contains a `tool_drift` entry listing the tool differences

#### Scenario: Same tool with different parameters
- **WHEN** both packs call the same tool but with different parameter values
- **THEN** the drift report contains a `param_drift` entry showing the parameter differences

#### Scenario: Intermediate outputs diverge
- **WHEN** a step at the same index produces different output in each pack
- **THEN** the drift report contains a `reasoning_drift` entry identifying the step index and output hashes

#### Scenario: Final outputs differ
- **WHEN** the output artifacts differ between the two packs
- **THEN** the drift report contains an `output_drift` entry listing the differing outputs

### Requirement: Machine-readable diff output
The default diff output format SHALL be JSON, suitable for programmatic consumption.

#### Scenario: Default output is JSON
- **WHEN** the user runs `ctx diff <a> <b>` without format flags
- **THEN** the output is valid JSON containing an array of typed drift entries

### Requirement: Human-readable diff output
The system SHALL support a `--human` flag that produces a human-readable summary of decision points that changed between two packs.

#### Scenario: Human-readable output
- **WHEN** the user runs `ctx diff <a> <b> --human`
- **THEN** the output is a plain-text summary describing each drift point in natural language

#### Scenario: No drift in human mode
- **WHEN** the user runs `ctx diff <a> <a> --human`
- **THEN** the output states that no differences were found

### Requirement: Step alignment for comparison
When comparing steps between two packs, the system SHALL align steps by index position. Steps present in one pack but not the other SHALL be reported as additions or removals.

#### Scenario: Pack B has more steps than pack A
- **WHEN** pack A has 5 steps and pack B has 7 steps
- **THEN** the drift report compares the first 5 steps by index and reports steps 6-7 as additions in pack B

#### Scenario: Pack A has more steps than pack B
- **WHEN** pack A has 7 steps and pack B has 5 steps
- **THEN** the drift report compares the first 5 steps by index and reports steps 6-7 as removals in pack B
