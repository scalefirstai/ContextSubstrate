## ADDED Requirements

### Requirement: Replay a context pack
The system SHALL provide a `ctx replay <hash>` command that re-executes an agent run step-by-step as recorded in the context pack.

#### Scenario: Replay a pack with all tools available
- **WHEN** the user runs `ctx replay <hash>` and all tools referenced in the pack are available
- **THEN** the system re-executes each step in order and produces a replay report

#### Scenario: Replay a non-existent pack
- **WHEN** the user runs `ctx replay <hash>` with a hash not found in the store
- **THEN** the system exits with non-zero status and reports the pack was not found

### Requirement: Step-by-step execution comparison
During replay, the system SHALL re-execute each step in the recorded order and compare the actual output hash against the recorded output hash for that step.

#### Scenario: Step produces identical output
- **WHEN** a replayed step produces output with the same SHA-256 hash as the recorded output
- **THEN** the step is marked as "matched" in the replay report

#### Scenario: Step produces different output
- **WHEN** a replayed step produces output with a different SHA-256 hash than the recorded output and the step is marked as deterministic
- **THEN** the step is marked as "diverged" in the replay report with both expected and actual hashes

### Requirement: Replay fidelity reporting
The system SHALL report replay fidelity at three levels: `exact` (all deterministic step outputs match), `degraded` (some deterministic steps diverged but execution completed), and `failed` (a step could not execute).

#### Scenario: All steps match
- **WHEN** replay completes and every deterministic step's output matches the recorded hash
- **THEN** the replay report status is `exact`

#### Scenario: Some steps diverge
- **WHEN** replay completes but one or more deterministic steps produced different output hashes
- **THEN** the replay report status is `degraded` and the divergent steps are listed

#### Scenario: A step cannot execute
- **WHEN** a step references a tool that is not available or an input that cannot be resolved
- **THEN** the replay report status is `failed` and the failing step is identified with the reason

### Requirement: Non-deterministic step handling
Steps marked as `"deterministic": false` in the pack manifest SHALL be expected to diverge during replay. Their divergence SHALL NOT downgrade the replay fidelity level.

#### Scenario: Non-deterministic step diverges
- **WHEN** a step marked `"deterministic": false` produces different output during replay
- **THEN** the step is noted as "diverged (expected)" and the overall fidelity is NOT downgraded

#### Scenario: Non-deterministic step matches
- **WHEN** a step marked `"deterministic": false` produces identical output during replay
- **THEN** the step is marked as "matched" with no special treatment

### Requirement: Dependency drift identification
When replay fidelity is `degraded` or `failed`, the replay report SHALL identify which specific dependencies drifted (tool version changes, missing inputs, environment differences).

#### Scenario: Tool version changed
- **WHEN** a tool used during replay has a different version than recorded in the pack's environment metadata
- **THEN** the replay report includes a drift entry identifying the tool and the version mismatch

#### Scenario: Input file missing
- **WHEN** replay requires an input file that cannot be resolved from the object store
- **THEN** the replay report includes a drift entry identifying the missing input with its expected hash
