## 1. Implementation
- [x] 1.1 Validate the change with `openspec validate fix-accel-session-concurrency --strict`
- [x] 1.2 Write the concurrency tests first — concurrent `ExecuteDataList` plus concurrent readers over every public method — and confirm `go test -race` fails against the current code
- [x] 1.3 Add the session mutex: public methods lock, internal callers switch to unexported `*Locked` bodies (`reportLocked`, `cacheSnapshotLocked`, `planShardableWorkloadLocked`)
- [x] 1.4 Lock `Discover`, `RegisterDevice`, `RecordReport`, `Close`, `Closed`, `Config`, `Devices`, `Reports`, `LastReport`, and the cache-insertion tail of `ProjectDataList`/`ProjectDataTable`
- [x] 1.5 Serialize `internal/wgpu.Sum` behind a package-level mutex
- [x] 1.6 Confirm the tests from 1.2 pass under `-race`, then run the full suite, and the GPU test with `INSYRA_ACCEL_GPU_TESTS=1`
- [x] 1.7 Update CHANGELOG and CHANGELOG_TW, delete the resolved follow-up from `AGENTS.md`, and update `delivery-plan.md`
- [x] 1.8 Skip device-touching tests and benchmarks under the `race` build tag, and record the upstream `checkptr` abort in `AGENTS.md`
