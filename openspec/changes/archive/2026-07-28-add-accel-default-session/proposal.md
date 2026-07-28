# Change: Provide a process-shared default session

## Why
Every caller that wants acceleration has to build and own a `Session`. That is fine for a program calling `accel` directly, and wrong for the roadmap's end state, where library code inside `stats` or the CCL evaluator consults acceleration on behalf of a user who never mentioned it. Those call sites cannot each open a session: discovery would run per call, every session would keep its own cache so nothing would ever stay resident across operations, and nobody would own `Close`.

A single lazily-created session shared by the process solves all three. It became viable only once `Session` was made safe for concurrent use.

## What Changes
- Add `accel.Default()`, returning a lazily created session shared by the process
- Run discovery once, on first use, rather than at package initialisation
- Make the shared session's `Close` a no-op, since no single caller owns it
- Add `accel.ResetDefaultForTest()` so tests can force re-discovery

## Impact
- Affected specs: `accel-runtime`
- Affected code: new `accel/default.go`
