# AGENTS.md (nn/)

Rules for working in `nn/`. The top-level [AGENTS.md](/AGENTS.md) covers the wider repo.

## Known pitfalls

- `nn`'s tests import `ml` (`ml_protocol_roundtrip_test.go`, `protocol_test.go`, `roundtrip_test.go`), so a half-finished edit in `ml` breaks `go test ./nn/`; run parallel `ml` and `nn` work in separate git worktrees.
- Do not propose changing the worker count of `matMul2DWithWorkers` (`nn/kernels.go`) as a speed fix: the `pthread_cond_wait`/`signal` samples beside it in a CPU profile are scheduler idle and wake-up, and scaling the workers to the work changed 16 training shapes by 0.95-1.11x. Benchmark on a quiet host with alternating old/new runs and best-of-N.
