## ADDED Requirements

### Requirement: Every main spec passes strict validation

Every spec under `openspec/specs/` SHALL pass `openspec validate --specs --strict`: it SHALL have a `## Purpose` of at least 50 characters that says what the capability is for, and every requirement's text before its first scenario SHALL be at most 500 characters, with at least one scenario.

#### Scenario: Validating the specs
- **WHEN** `openspec validate --specs --strict` runs on the branch
- **THEN** it reports every spec as valid and exits 0

### Requirement: CI runs the strict validation

The GolangCI-Lint workflow SHALL run `openspec validate --specs --strict` with a pinned OpenSpec version on every push and pull request to `main`, `dev` and `0.4`, and SHALL fail when any main spec fails it.

#### Scenario: A spec with a placeholder Purpose
- **WHEN** a commit archives a change that creates a capability and leaves `TBD - created by archiving change …` as its Purpose
- **THEN** the workflow fails on that commit and names the spec

### Requirement: Splitting a requirement keeps its meaning

When a requirement is split to meet the length limit, every rule it stated SHALL survive in one of the resulting requirements, every one of its scenarios SHALL stay, and the first resulting requirement SHALL keep the original header so a later `MODIFIED` delta still finds it.

#### Scenario: A long requirement is split
- **WHEN** a requirement of more than 500 characters is divided into several
- **THEN** the first keeps the original header, each part has at least one scenario, and no rule or scenario is dropped
