## Context

Acceleration work spans planning, repository instructions, runtime packages,
CLI exposure, and several backend proposals. Before this change those decisions
lived in agent conversations, so a handoff could not reliably tell which phase
was active, which change was next, or what evidence was still missing.

This change is deliberately a control-surface change. It creates the shared
delivery plan, the durable agent contract, the bootstrap pointer, and the
proposal inventory before runtime implementation is treated as in scope.

## Goals / Non-Goals

**Goals:**

- Make phase, milestone, blocker, next output, and handoff state durable.
- Make accel-specific planning and validation rules discoverable by agents.
- Keep the full Phase 0 and Phase 1 proposal inventory in the repository.
- Separate planning decisions from runtime implementation commits.

**Non-Goals:**

- Adding a GPU backend, kernel, scheduler, or CLI behavior.
- Replacing the repository's general OpenSpec workflow.
- Freezing every future performance threshold or hardware result.

## Decisions

### Use `delivery-plan.md` as the shared progress surface

The plan records the current phase, one verifiable milestone per change,
blockers, decision deltas, source links, and handoff notes. It is updated after
each accel milestone so the next agent can continue from evidence rather than
reconstructing history.

### Put operating rules in `AGENTS.md` and keep `CLAUDE.md` as a pointer

`AGENTS.md` is the authoritative contract for accel work. `CLAUDE.md` only
bootstraps tools that read Claude-specific instructions and points back to the
same source. Duplicating the contract was rejected because the two copies would
drift during a long acceleration effort.

### Map every runtime slice to one OpenSpec change

Each active accel item has one named change and one verifiable output. Umbrella
proposals are excluded so a review can connect a measurement and implementation
to one requirement set and one rollback boundary.

## Risks / Trade-offs

- **[Risk] The plan becomes stale]** → Treat milestone, blocker, handoff, and
  next-change updates as part of the milestone itself.
- **[Risk] Agents bypass the contract for a small runtime edit]** → Keep the
  entry sequence explicit and require a named change before implementation.
- **[Risk] The proposal inventory expands faster than implementation]** → Use
  the inventory for scope and ordering only; each item still needs its own
  validation before entering execution.

## Migration Plan

No runtime migration is required. Future accel work reads the plan and
`AGENTS.md` before implementation, validates its named change, and updates the
handoff surface after each milestone. Reverting this change returns planning to
chat history and is therefore not recommended, but it does not alter library
behavior.

## Open Questions

The runtime changes, backend choice, and phase boundaries remain decisions for
their individual OpenSpec changes. This control-surface change intentionally
does not decide them a second time.
