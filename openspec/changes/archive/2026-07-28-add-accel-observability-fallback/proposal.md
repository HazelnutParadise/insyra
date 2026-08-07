# Change: Define acceleration observability and fallback reporting

## Why
The accel phase explicitly chose observable fallback rather than silent fallback. That requires a spec for reason codes, execution reports, and user-visible metrics before implementation starts.

## What Changes
- Add a new `accel-observability` capability
- Define fallback reason codes and execution reports
- Define cache and device usage metrics
- Define strict-vs-auto mode output behavior

## Impact
- Affected specs: `accel-observability`
- Affected code: future reporting surface, CLI/DSL integration, and runtime metrics emission
