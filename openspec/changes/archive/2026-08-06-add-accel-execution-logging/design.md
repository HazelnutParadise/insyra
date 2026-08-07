# Design: add-accel-execution-logging

## Context

`accel` reports execution outcomes through `ExecutionResult` and session reports, but writes nothing to the console. The root package has a structured, level-controlled logger used across the library. The session is the natural once-per scope: it owns discovery, the eligible device set, and the report history.

## Goals / Non-Goals

**Goals:**

- A user running with defaults sees one line saying acceleration engaged (device, backend, mode, strategy) and, if it ever falls back, one line saying why.
- Debug level exposes per-execution placement without recompiling.
- Zero noise growth with call count.

**Non-Goals:**

- No new report fields, no API changes, no logging in the hot per-row paths.
- No logging at `Open`/discovery time — discovering a device is not using it; the honest event is first use.

## Decisions

1. **Log at first use, not at discovery.** The info line fires when the first operation actually executes on a device, so it never claims acceleration that never happened.
2. **Once-per-session flags, atomically guarded.** The session already serializes state; two booleans (first-use logged, first-fallback logged) keep the guarantee under concurrent use, which the session explicitly supports.
3. **First fallback logs at info only when a device could have run** — sessions on hosts with no eligible device already said so via the selection/discovery path; repeating it per operation is noise. Ineligible-by-caller's-own-terms requests (explicit algorithm, below floors) stay debug-only, matching the operating contract's distinction.
4. **Everything routes through the root logger** so `Config` log levels govern acceleration exactly like the rest of the library.

## Risks / Trade-offs

- [Log lines in libraries annoy some users] → one line per session at info, silenceable by the existing log-level control; detail lives at debug.
- [Concurrent sessions interleave lines] → each line carries the device/session identity; no cross-session state.

## Open Questions

- None.
