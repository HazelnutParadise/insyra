# Change: Make accel.Session safe for concurrent use

## Why
`Session` holds `devices`, `reports`, and `cache` with no lock of any kind — neither `session.go` nor `cache.go` imports `sync`. `ExecuteProjectedDataset`, `cacheDataset`, `recordExecutionMetrics`, and `Discover` all write those fields, so two goroutines sharing one session race on plain maps and slices. The library's stated default is that thread safety is on via the actor model, and the accepted roadmap makes it worse than a documentation gap: the end state is a process-shared default session consulted implicitly from library code, which is exactly the usage pattern the current code cannot survive. The race detector stays green today only because every accel test is single-goroutine.

The GPU handle underneath has the same exposure one level down: two sessions share one `internal/wgpu` device handle, and gogpu's own queue-concurrency guarantees are unverified.

## What Changes
- Serialize all `Session` state behind an internal mutex; public methods lock, internal helpers assume the lock is held
- Serialize device submission inside `internal/wgpu`, so concurrent sessions cannot interleave gogpu calls
- Add always-on concurrency tests that fail under `-race` against the old code
- Remove the resolved follow-up from `AGENTS.md`

## Impact
- Affected specs: `accel-runtime`
- Affected code: `accel/session.go`, `accel/discovery.go`, `accel/planner.go`, `accel/executor.go`, `accel/cache.go`, `accel/dataset.go`, `accel/internal/wgpu/wgpu.go`
