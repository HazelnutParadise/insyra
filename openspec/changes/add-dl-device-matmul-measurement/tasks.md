# Tasks

## 1. Prototype and harness

- [x] 1.1 Tiled f32 WGSL matmul prototype reachable only from `accel`'s test/benchmark surface (build-tag or _test.go file), gated on `INSYRA_ACCEL_GPU_TESTS=1` like the existing device tests
- [x] 1.2 Benchmark harness timing device (upload+dispatch+readback) vs the all-core CPU path at the hot shapes from the proposal, best-of-5 each

## 2. Measurement

- [x] 2.1 Run on the 8-core M3 / Metal host; record per-shape device vs CPU wall time
- [x] 2.2 Record per-shape max-absolute and ULP deviation device vs CPU
- [x] 2.3 Write the verdict into `delivery-status.md`: decision entry with numbers, M17 milestone row updated (in progress → scoped ticket, or closed negatively)

## 3. Sync

- [x] 3.1 No user-visible change — no changelog entry. `AGENTS.md` follow-ups updated only if the verdict creates or resolves one.
