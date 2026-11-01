# Tasks

- [x] `accel/internal/wgpu/warmup.go` (`builtinKernels`, `startWarmupLocked`, `warm`, `warmMu`, `warmInFlight`), `kernel.compile`/`install` in `resident.go`, `releaseLostDevice` in `wgpu.go`.
- [x] Tests: every package kernel is listed; a buffer starts warmup and warmup builds every kernel; releasing a lost device releases the in-flight kernel; a warmup whose device was released stops.
- [x] Measured: warm-cache first steps equal `dev` on the M3 and the GT 1030; the GPU suites pass under `-race` on the M3 and on the GT 1030.
- [x] `Docs/accel.md` and both changelogs.
- [x] `openspec validate accel-kernel-warmup --strict` passes.
