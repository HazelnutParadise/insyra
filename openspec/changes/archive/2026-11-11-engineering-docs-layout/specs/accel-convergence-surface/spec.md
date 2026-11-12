## MODIFIED Requirements

### Requirement: Shared acceleration control surface
The project SHALL provide a shared control surface for acceleration work through `AGENTS.md`, `CLAUDE.md`, the engineering documents under `engineering/` and the open changes under `openspec/changes/`.

#### Scenario: New agent enters the repository
- **WHEN** a new agent starts accel-related work
- **THEN** it can determine the open work and what each piece waits for by reading the open changes' `proposal.md` files
- **AND** it can determine why a standing choice was made by reading the records under `engineering/decisions/`
- **AND** it can determine unfinished work that no change covers by reading `engineering/handoff.md` when that file exists
- **AND** it can determine the project operating rules by reading `AGENTS.md`
- **AND** `CLAUDE.md` points it to `AGENTS.md`

### Requirement: Explicit handoff continuity
The control surface SHALL capture handoff continuity without relying on prior chat history.

#### Scenario: Milestone or blocker changes
- **WHEN** a milestone status changes, a blocker appears, or work is handed off
- **THEN** the open change the work belongs to records the state, the blocker and the next verifiable output, or `engineering/handoff.md` records them when no change covers the work
- **AND** a decision that holds beyond that change is written as a new record under `engineering/decisions/`
