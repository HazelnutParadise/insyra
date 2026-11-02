# Verification evidence

## Environment

- Initial dev base: `99877e62`; Go `go1.26.9 darwin/arm64`; golangci-lint `2.12.2`.
- Real resident device probe: `device=Apple M3 backend=metal error=<nil>`.

## Tests first

- `go test ./nn -run TestOptimizerClear -count=1` failed with the no-op lifecycle skeleton: Clear policy was not retained and freeze was not retained in all eight exact/rounded optimizer combinations.
- `go test ./nn -run OptimizerState -count=1` failed with the export/import skeleton: every host subprocess continuation combination and the import fixture setup reported `optimizer state export is not implemented`.
- `INSYRA_ACCEL_GPU_TESTS=1 go test ./nn -run '^TestOptimizerStateDeviceRestart$' -count=1 -v` ran on Metal and failed at the same unimplemented export for both Adam and AdamW.
- The executable resume example extracted from Docs/nn.md failed at the unimplemented export before persistence implementation.
- The independent half-wire fixture check initially failed because its scalar header used `shape: null`. The test helper now writes `shape: []`; its validation check passes before persistence implementation. Numeric corruption helpers preserve original shapes so a shape error cannot masquerade as non-finite-state rejection.

## Verified lifecycle behavior

`INSYRA_ACCEL_GPU_TESTS=1 go test ./nn -run 'TestOptimizerClear|TestFrozenParameter' -count=1 -v` passes, including retained resident gradients, aliased operation-output gradients with one owner, Close returning resident bytes to baseline, frozen momentum/decay, and caller-managed Update on a frozen parameter.

Three additional Metal tests first failed with `wgpu: buffer was released`: promoting a retained output gradient to a parameter, reusing it in a later successful reverse pass, and preserving it after a failed pass. Backward cleanup now excludes tensors still owned by the graph, a successful gradient pass or a parameter. All three pass, including resident-byte checks after Clear/Close. An independent adversarial review found no remaining confirmed defect after this fix.

## dev checks

- `go build ./...` and `go vet ./...`: pass.
- `go test -json ./nn/...`: pass, with 573 test/subtest passes and 61 skips. The ordinary run skips optional hardware, Python/reference, data and exhaustive gates; the relevant gates below ran separately.
- `INSYRA_ACCEL_GPU_TESTS=1 go test ./nn -run 'Test(RetainedGradient|FailedBackwardKeepsRetained)|OptimizerState|TestOptimizerClear|TestFrozenParameter' -count=1 -v`: pass, covering all four issue acceptance conditions on the supported host/device paths.
- A broader Metal regression covering existing Custom/Resident paths, failed Backward and new persistence/lifecycle paths: pass.
- The same targeted host/Metal race-detector command recorded under the 0.4 checks also passes on dev.
- With the existing cross-language venv on PATH and `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1`, `TestSafeTensorsEmptyPythonReference` and `TestLoadSafeTensorsReferenceRoundTrip`: pass. Python writes and reads the empty/scalar F32/I64/BOOL round trip.
- The extracted Docs/nn.md resume example builds, runs, resumes and compares both weight/state byte streams successfully.
- `golangci-lint run`, using a task-specific cache: 0 issues. The first shared-cache run reported stale paths from another worktree; an isolated run exposed two local findings (the reflect.Ptr alias and an unused release helper), both corrected before the clean run.

The available hardware is Apple M3/Metal. This does not establish Vulkan, DirectX or multi-card verification. Momentum remains host-only, matching its existing implementation.

## 0.4 port checks

Ported from dev commit `4c9c3049` onto `87bf1afe`. All new implementation and test files are byte-identical between lines. Existing 0.4 API changes remain in place; conflict resolution retained its changelogs, MNIST note and status structure. The default device MatMul implementation is the same on both lines, so its platform caveat is documented on 0.4 too.

- `go build ./...`, `go vet ./...`, and `golangci-lint run` with an isolated cache: pass, 0 lint issues.
- `go test -json ./nn/...`: pass, 559 test/subtest passes and 60 optional skips.
- The full targeted host/Metal persistence, Clear/freeze, retained-gradient and existing Custom/Resident regression: pass, including actual subprocess restart and allocation failure atomicity.
- `INSYRA_ACCEL_GPU_TESTS=1 go test -race ./nn -run 'OptimizerState|RetainedGradient|FailedBackwardKeepsRetained|OptimizerClear|FrozenParameter|SafeTensorsScalarAndEmpty' -count=1`: pass.
- Strict Python SafeTensors scalar/empty and original reference round trips: pass.
- Executable resume example: pass, next-step weights and optimizer-state byte streams equal.
- `openspec validate --specs --strict`: 236 passed, 0 failed.

## Requirement coverage and review

| Requirement | Implementation | Acceptance evidence |
| --- | --- | --- |
| Named binary persistence | `nn/optimizer_state.go`, SaveOptimizerState / LoadOptimizerState | Six host subprocess combinations, two Metal subprocess combinations, host/device transfer, mixed slots, name-order independence, signed zero/subnormals and counters above 2^53 |
| Transactional validation | All validation and resident staging precede assignment-only commit | Valid-SafeTensors corruption matrix, malformed files, invalid mappings, late allocation failure with no mutation/leak |
| Bounded checkpoint input | `nn/optimizer_state_reader.go`, shared SafeTensors data decoder | Counting poison readers, header-only rejection, one-byte tail probe, final-byte I/O error, long-name/high-rank empty writer output |
| Explicit Clear policy | `nn/optimizer_lifecycle.go` and protected Backward cleanup | Repeated cycles, gradient replacement/retention, promoted/reused/shared gradients, failed reverse pass and resident bytes returning to baseline |
| Frozen built-in updates | SGD, momentum, Adam and rounded Adam loops | Exact/rounded comparisons of values, velocity, moments and steps; device AdamW; explicit caller Update remains available |

Implementation/specification review covers all five requirements and the four issue acceptance conditions. A fresh reader reviewed the complete nn documentation and change artifacts after corrections and found no remaining readability issue. An independent adversarial reviewer rechecked the ownership fix and ran the broader Metal regressions with no remaining confirmed finding. No public name or signature was replaced, so no deprecation/removal or BREAKING entry is needed. The nn documentation location and the principles taught by the distributed agent skill are unchanged; API details remain in Docs.

## Specification baseline

`openspec validate save-nn-optimizer-state --strict` passes. Before adding the new main capability, `openspec validate --specs --strict` reports 213 passed, 0 failed.

## External references checked

- [SafeTensors format](https://github.com/safetensors/safetensors#format): binary little-endian tensors, row-major layout, scalar and empty tensors, duplicate key restrictions.
- [`math.Float32bits`](https://pkg.go.dev/math#Float32bits): binary representation and inverse conversion; also checked with local `go doc math.Float32bits`.
- [`base64.RawURLEncoding`](https://pkg.go.dev/encoding/base64#RawURLEncoding): unpadded alternate encoding; also checked with local `go doc encoding/base64.RawURLEncoding`.
- Local `openspec archive --help`: `--yes` archives and syncs specs without prompting, as requested for this PR.

## PR delivery

- [dev PR #674](https://github.com/HazelnutParadise/insyra/pull/674) targets `dev`.
- [0.4 PR #673](https://github.com/HazelnutParadise/insyra/pull/673) targets `0.4`.
- Both bodies include Before/After/How, the API and executable resume example, checked external references, and the statement that #376 can close once both merge. The owner has authorized merging once CI is all green and review findings are addressed; the PR links record the live gate and merge status.

## PR review corrections (2026-11-02)

- Tests were added before implementation. The original loader consumed the 1 MiB poison suffix for oversized headers, unknown keys, wrong slot shapes and trailing input; the ordinary whitespace/metadata/scalar/empty control passed. On Metal, the new live-graph-input case failed after the first reverse pass with `wgpu: buffer was released`.
- Imports now bound header allocation from registered names/ranks, validate allowed wire descriptors and offsets before payload reads, then read the validated payload and one trailing byte at most. The existing decoder consumes the buffered payload directly. An observing reader preserves non-EOF errors even when ReadFull fills its destination on the same read. Long Unicode names and high-rank empty parameters still round-trip.
- Backward cleanup transfers old gradient buffers still referenced by a live graph into graph ownership. Repeated reverse passes succeed, a later Clear releases the obsolete input, and Clear/Close return resident bytes to baseline.
- On both lines, `go build ./...`, `go test ./nn/...`, `go vet ./...` and `golangci-lint run` pass; lint uses separate task caches and reports 0 issues. Targeted host/Metal persistence, lifecycle and freeze tests, the race-detector command above, and strict Python SafeTensors reference round trips pass. All main specs pass strictly (214 on dev, 236 on 0.4).
- The new reader, import integration, lifecycle implementation and regression files are byte-identical between lines. The 0.4 changelog conflict was resolved by preserving its existing content and changing only the optimizer import sentence. Its existing API and codec differences are retained.
- An independent fresh-context security/compatibility review inspected the candidate and its callers, ran CPU and verbose real-device regressions, and found no remaining confirmed finding or documentation/specification readability gap. No public signature or wire version changed.
- Checked read semantics with local `go doc io.ReadFull`; [official io.ReadFull documentation](https://pkg.go.dev/io#ReadFull).
