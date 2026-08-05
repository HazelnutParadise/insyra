# Design: add-accel-device-selection

## Context

`Config` already carries `Mode` (auto/cpu/gpu/strict-gpu), `Strict`, `EnableFallback`, `PreferredBackends`, and the soft `PreferredDevices`. Devices have stable string IDs and a discovery order. The `INSYRA_ACCEL_STUB_*` probes can fabricate arbitrary device lists, so every selection rule is testable on this host without hardware.

## Goals / Non-Goals

**Goals:**

- Hard selection exists at both the ops level (env) and the program level (config), with the DL-ecosystem semantics people already know.
- The three axes — execution mode, device bounds, soft preference — compose predictably, and every combination's behavior is a tested rule rather than an emergent accident.
- Defaults change nothing.

**Non-Goals:**

- No per-call device pinning: sessions are the placement scope, as they are today. (PyTorch's per-tensor placement answers a tensor-graph problem this columnar library does not have; considered and declined.)
- No renaming or deprecation of `PreferredDevices`.
- No scheduling changes — this change decides *which* devices are eligible, not *how many* run one operation.

## Decisions

1. **Mask at the discovery boundary.** `INSYRA_ACCEL_DEVICES` filters the device list before anything downstream sees it — masked devices do not exist, the exact semantics of `CUDA_VISIBLE_DEVICES`. This keeps every later layer (planner, executor, reporting) ignorant of masking rather than each consulting a policy.
2. **Config allowlist intersects, not overrides.** Eligible = env ∩ config. The env var is the operator's boundary; a program cannot reach outside it, matching container-orchestration expectations. Disjoint sets leave an empty eligible set, handled per mode.
3. **Accept device IDs and zero-based discovery indices.** IDs are exact and stable; indices (`0`, `1`) are what `CUDA_VISIBLE_DEVICES` users type. Both forms, comma-separated in the env var, either form in `Config.Devices`. An entry matching nothing is recorded in the session report rather than silently ignored — a typo that quietly widens eligibility is the failure mode to prevent.
4. **Empty eligible set behaves per mode.** Strict modes (`ModeStrictGPU`, `Strict: true`) return an error naming the bound that emptied the set; automatic modes fall back to CPU with a dedicated `FallbackReason` so the outcome is observable, consistent with "a missing device is a performance event".
5. **Soft preference survives unchanged, scoped to the eligible set.** `PreferredDevices` orders what selection and planning consider — it can no longer resurrect a masked device.

## Risks / Trade-offs

- [Two hard bounds confuse users] → the doc section shows the three axes as a table with the intersection rule; the session report lists the eligible set so what happened is inspectable.
- [Index form is order-dependent] → discovery order is already deterministic per host; the doc says indices are host-specific and IDs are the portable form.
- [Unmatched entries] → reported, not fatal in automatic modes; fatal in strict modes only when the eligible set ends up empty.

## Open Questions

- None blocking.
