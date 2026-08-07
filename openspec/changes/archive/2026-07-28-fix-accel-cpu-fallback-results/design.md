## Context

The acceleration API can discover that no device ran, but its distance and
nearest-query entry points previously returned empty result slices in that
case. The CPU reference already existed, so the runtime had all the information
needed to answer while incorrectly making callers branch on `Accelerated` for
correctness.

The design must preserve the distinction between a device being unavailable and
the request itself being ineligible. A caller that asked for exact precision or
an unsupported dtype has not authorized the runtime to silently answer under a
different contract.

## Goals / Non-Goals

**Goals:**

- Return the CPU answer when device discovery, registration, profitability, or
  execution prevents acceleration.
- Preserve `Accelerated` and `FallbackReason` as observability fields.
- Use the same CPU reference for no-device and device-error paths.
- Keep request-side ineligibility returning no result.
- Keep strict GPU mode failing instead of falling back.

**Non-Goals:**

- Hiding why the device did not run.
- Making an ineligible request eligible by changing precision or dtype.
- Adding a new accelerator or changing the exact-nearest algorithm.
- Treating an empty answer as a valid result for zero-input error cases.

## Decisions

### Classify fallback reasons by who caused the fallback

A small predicate separates device-side reasons, such as no accelerator,
discovery failure, unprofitable work, buffer or shader failure, timeout, and
execution failure, from request-side reasons such as rejected precision,
unsupported dtype, or unsupported workload. Only the device-side class invokes
the CPU reference.

This is safer than falling back on every `Accelerated == false` result because
the latter would silently reinterpret the caller's explicit terms.

### Reuse the exported CPU references

`ExecuteDistances` and `ExecuteNearestQuery` call the existing CPU reference
with the original numeric inputs when a device-side reason is present. The
returned result contains the exact indices and distances produced by that
reference, while the execution metadata records that acceleration did not
occur.

Maintaining a second fallback loop was rejected because it could drift from the
reference and make device absence a correctness concern again.

### Preserve strict mode as an explicit exception

Strict GPU mode returns the device or execution error and does not substitute a
CPU answer. Strict mode exists for callers that require device execution, so
changing it would make the mode name and failure contract misleading.

## Risks / Trade-offs

- **[Risk] A new fallback reason is misclassified]** → Add it to the predicate
  with a test that asserts both result and reason, and review whether the
  caller's terms or device state caused it.
- **[Risk] CPU fallback hides a serious device regression]** → Keep the reason
  and `Accelerated: false` observable and retain device-path tests separately.
- **[Risk] CPU work is slower than a caller's old manual branch]** → The default
  contract prioritizes a correct answer; callers that need strict performance
  can inspect the metadata or use strict mode.
- **[Risk] Strict mode accidentally falls through to the CPU]** → Test strict
  discovery and execution failures independently.

## Migration Plan

This changes the result behavior for callers that previously received empty
slices after a device-side fallback. They can remove manual CPU branches and
continue inspecting metadata for performance reporting. Request-side
ineligibility and strict mode keep their existing behavior. No persisted data or
API migration is required.

## Open Questions

Future accelerator operations should reuse the same reason classification and
CPU-reference rule. If an operation cannot provide a complete CPU reference,
its OpenSpec change must define that exception before adding a fallback path.
