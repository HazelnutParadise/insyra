# accel-convergence-surface Specification

## Purpose
The single place a program turns acceleration on and off and reads where its work ran, so every accelerated operation in the module shares one control surface; it also holds the planning rules that keep accel work covered by an OpenSpec change and handed off with its state.
## Requirements
### Requirement: Shared acceleration control surface
The project SHALL provide a shared control surface for acceleration work through `AGENTS.md`, `CLAUDE.md`, the engineering documents under `engineering/` and the open changes under `openspec/changes/`.

#### Scenario: New agent enters the repository
- **WHEN** a new agent starts accel-related work
- **THEN** it can determine the open work and what each piece waits for by reading the open changes' `proposal.md` files
- **AND** it can determine why a standing choice was made by reading the records under `engineering/decisions/`
- **AND** it can determine unfinished work that no change covers by reading `engineering/handoff.md` when that file exists
- **AND** it can determine the project operating rules by reading `AGENTS.md`
- **AND** `CLAUDE.md` points it to `AGENTS.md`

### Requirement: Full proposal coverage before accel execution
The project SHALL create the full OpenSpec proposal inventory for the active acceleration phase before runtime implementation starts.

#### Scenario: Phase kickoff for accel work
- **WHEN** the acceleration phase begins
- **THEN** all active stage items map to named OpenSpec changes
- **AND** each change maps to one milestone and one verifiable output
- **AND** the next agent can start with one named change without re-planning the phase

### Requirement: Explicit handoff continuity
The control surface SHALL capture handoff continuity without relying on prior chat history.

#### Scenario: Milestone or blocker changes
- **WHEN** a milestone status changes, a blocker appears, or work is handed off
- **THEN** the open change the work belongs to records the state, the blocker and the next verifiable output, or `engineering/handoff.md` records them when no change covers the work
- **AND** a decision that holds beyond that change is written as a new record under `engineering/decisions/`

