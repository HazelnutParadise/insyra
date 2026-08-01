## Context
The repo already has CLI and DSL specifications. Acceleration should extend those surfaces in a way that makes backend choice and fallback visible without forcing users to drop into internal APIs.

## Goals / Non-Goals
- Goals:
  - expose accel devices, cache state, and execution modes through CLI and DSL
  - keep accel observability visible from user-facing entry points
  - preserve existing CLI/DSL structure instead of inventing a second shell
- Non-Goals:
  - implementing all runtime features in this change
  - exposing every backend tuning flag in v1

## Decisions
- Decision: use an `accel` command group in CLI
  - Rationale: it groups device, cache, and mode concerns without overloading existing commands.
- Decision: expose mode selection in both CLI and DSL/config
  - Rationale: accel behavior must be consistent across scripts, REPL, and command-line use.
- Decision: include selected backend, selected devices, and fallback reasons in user-visible output
  - Rationale: observable fallback is part of the contract, not an internal trace only.

## Risks / Trade-offs
- Risk: user-facing output could become noisy.
  - Mitigation: keep default summaries concise and leave detailed reports to structured subcommands.
