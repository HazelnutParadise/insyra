## ADDED Requirements

### Requirement: Shared acceleration control surface
The project SHALL provide a shared control surface for acceleration work through `delivery-plan.md`, `AGENTS.md`, and `CLAUDE.md`.

#### Scenario: New agent enters the repository
- **WHEN** a new agent starts accel-related work
- **THEN** it can determine the current phase, blockers, next verifiable output, and next OpenSpec change by reading `delivery-plan.md`
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
- **THEN** `delivery-plan.md` is updated with the new phase state, next verifiable output, next OpenSpec change, and decision delta
