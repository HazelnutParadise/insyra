## Context
Acceleration mode is only safe as a default if the user can inspect what happened. The runtime therefore needs structured reporting for backend choice, fallback reasons, and device/cache activity.

## Goals / Non-Goals
- Goals:
  - define fallback reason codes
  - define user-visible execution reports
  - define metrics for cache and device usage
- Non-Goals:
  - full telemetry backend integration
  - prescribing final output formatting for every UI surface

## Decisions
- Decision: fallback must be reportable with stable reason codes
  - Rationale: downstream CLI/DSL and tests need predictable output semantics.
- Decision: selected backend and selected devices are first-class report fields
  - Rationale: users need to know not just that acceleration ran, but where it ran.
- Decision: strict and auto modes produce different failure behavior but compatible reports
  - Rationale: output should remain inspectable even when execution policy differs.

## Risks / Trade-offs
- Risk: exposing too many metrics may create an unstable reporting contract.
  - Mitigation: define a minimal stable core set and reserve extended metrics for later changes.
