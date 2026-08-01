## Context

Phase 1 is establishing the acceleration runtime, device discovery, residency,
fallback, and scheduling contracts. Full string-kernel parity is a different
problem: it needs a representation for variable-width values, device-side
encoding and comparison rules, and kernels whose performance depends on the
operation. Treating it as a prerequisite would make the runtime boundary
unreviewable and would turn an eligibility question into an accidental promise.

The existing plan already allows encoded strings and key-based operations to
be represented in the Phase 1 transport. This change records that boundary as a
separate capability and does not add a device implementation.

## Goals / Non-Goals

**Goals:**

- Make full string-kernel support an explicit Phase 2 capability.
- Keep Phase 1 runtime completion independent of full string-kernel parity.
- Preserve the distinction between encoded-string transport, key-based
  eligibility, and native string computation.
- Give later string-kernel work one proposal and specification surface to
  extend.

**Non-Goals:**

- Adding a string representation, WGSL kernels, or a new backend.
- Widening Phase 1 eligibility based only on this planning change.
- Changing the CPU fallback or the existing acceleration contracts.
- Claiming that encoded transport provides general string-operation support.

## Decisions

### Keep a separate OpenSpec capability

The Phase 2 capability has its own proposal and specification instead of being
folded into the runtime change. Runtime convergence can then be validated with
the narrower Phase 1 contract, while string work can define its own data model,
kernel set, and measurements.

An umbrella runtime proposal was rejected because it would make completion
depend on work that has different correctness and profitability criteria.

### Preserve encoded and key-based support as qualified Phase 1 support

The planning and eligibility surfaces may continue to carry encoded strings or
string keys where an existing operation explicitly supports them. The wording
must identify that as transport or key behavior, not as native string-kernel
parity. No new kernel is eligible merely because a value can be encoded.

Treating every encoded value as a string kernel was rejected because it would
hide lossy encoding and comparison semantics behind a type check.

### Use the delivery plan as the phase boundary

`delivery-plan.md` remains the shared source of truth for the current phase and
the next verifiable output. The change proposal and spec define what Phase 2
means; the delivery plan records when that work is admitted into the execution
sequence. This keeps handoffs from inferring phase scope from implementation
status or chat history.

## Risks / Trade-offs

- **[Risk] Encoded-string eligibility is mistaken for general string support]**
  → Keep the three cases named separately in the proposal, delivery plan, and
  eligibility documentation, and add a Phase 2 requirement before adding a
  kernel.
- **[Risk] The boundary becomes stale as a new operation lands]**
  → Require every future string-kernel proposal to state whether it changes
  Phase 1 eligibility or remains Phase 2, then validate the affected plan.
- **[Risk] Phase 2 is deferred indefinitely]**
  → Leave kernel requirements and measurements as an explicit next change,
  rather than allowing ad-hoc eligibility expansion in unrelated runtime work.

## Migration Plan

No runtime migration is needed. Keep existing Phase 1 transport and key-based
behavior unchanged, record the phase boundary in the planning artifacts, and
start a separate implementation change when a measured string workload and
device representation are available. Reverting this planning change only
removes the explicit boundary; it does not alter runtime code.

## Open Questions

Before Phase 2 implementation, decide the supported string representation,
null and encoding semantics, kernel inventory, and the minimum workload size at
which device execution is profitable. None blocks Phase 1 convergence.
