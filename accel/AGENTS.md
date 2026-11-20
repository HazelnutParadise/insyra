# AGENTS.md (accel/)

Rules for working in `accel/`. The top-level [AGENTS.md](/AGENTS.md) covers the wider repo.

## Known pitfalls

- Keep `isSoftwareAdapter` in `accel/internal/wgpu/wgpu.go` refusing adapters with `DeviceTypeCPU` or "software" in the name: gogpu registers a SPIR-V interpreter on every non-Android platform, and on a host with no GPU driver `RequestAdapter` returns it, so dropping the check reports a CPU interpreter as an acceleration device.
- With `INSYRA_ACCEL_GPU_TESTS=1`, run `go test ./accel/internal/wgpu` with `-timeout 60m` and alone on the machine: the default 10-minute timeout fails the package, and a run that shares the GPU with another suite is several times slower, so never read a slowdown from it.
- An `accel` test that asserts behaviour without a device sets `INSYRA_ACCEL_DISABLE_NATIVE_PROBES=1` itself; verify such tests one at a time on a GPU host (`go test -run '^TestX$'` per test), not only in a full-suite run, where an earlier test's reset can hide the dependence.
- In WGSL under `accel/internal/wgpu`, load an element of a struct-array parameter into a `let` before using it in an expression (`let x = a.d[i];`), never index it inline such as `(a.d[i] << 1u) | carry`: naga's Metal output then fails with "metal: failed to compile MSL: unknown error", and because every kernel includes the fixed library, every kernel breaks. Bisect such a failure by adding library functions one at a time against `TestExpAndLogRoundedMatchTheOracle`.
- Where Insyra's GPU API is behind industry practice (per-launch uniform and bind-group allocation instead of bind-once with a dynamic-offset parameter ring), fix it rather than documenting the gap, and design for ordinary discrete GPUs, not only Apple's; any claim about how other frameworks do it cites a source.
- When a bug is in an upstream GPU dependency (the `gogpu/*` and `go-webgpu/*` modules in `go.mod`), fix it with a PR upstream, not only an issue, and read the current dependency list from `go.mod` on the line you work on.
