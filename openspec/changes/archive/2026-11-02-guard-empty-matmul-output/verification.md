# Verification evidence

## Reproduction before implementation

- `TestEmptyMatrixOutputSizeErrors` failed on the original code: public/default MatMul, direct CPU workers, default tape and Gemm panicked on zero-payload F32/F16/BF16 inputs with `[MaxInt,0]` and `[0,2]`; the device hook was called six times. Rounded already rejected element-count overflow, but the separate 64-bit runtime-allocation-limit case panicked in MatMul, Gemm and rounded paths.
- `TestEmptyMatMulHugeShapes` failed in bounded child processes: default, rounded and Gemm hit the 10-second deadline; batched and vector-promoted cases panicked during batch-offset allocation. No truly endless test or physical OOM was used.
- `TestEmptyMatrixArithmeticAndShapes` passed before implementation, preserving the independent normal/empty control.

## Implementation and scope

Output multiplication is checked before the optional host-to-device hook. Default CPU MatMul and Gemm use the existing zero-tensor constructor, whose allocation expression converts runtime size-refusal panics to errors. Empty/zero-inner MatMul returns without row work or per-batch arrays, after vector/scalar metadata normalization. Gemm keeps Alpha*0 and C broadcasting. No public name, signature, dtype, SafeTensors format or precision changed; valid empty tensors are not rejected arbitrarily.

This does not promise recovery from fatal physical-memory exhaustion or introduce a RAM budget. The runtime-allocation-limit regression is gated to 64-bit, where the chosen count is deterministically beyond the runtime address-space limit without attempting physical allocation.

## dev verification

- `go build ./...`, `go test ./nn/...`, `go vet ./...`: pass.
- `golangci-lint run` with a task-local cache: pass, 0 issues.
- `go test ./nn -run 'TestEmptyMat|TestMatMul|TestGemmAndMatMul|TestBitIdenticalMatMul2D' -count=1`: pass, including element/runtime size errors, bounded huge-empty subprocesses, exact output shapes, all four rounded transpose cases and Gemm signed-zero/NaN/bias behavior.
- `INSYRA_ACCEL_GPU_TESTS=1 go test ./nn -run 'TestEmptyMat|OptimizerState|RetainedGradient|FailedBackwardKeepsRetained|OptimizerClear|FrozenParameter|Resident|Custom' -count=1`: pass on Apple M3/Metal.
- The new empty-matrix cases and persistence/lifecycle acceptance paths pass under the race detector. Strict Python SafeTensors scalar/empty/reference round trips pass.
- The separate resident path was inspected as a sibling; its reported `[MaxInt,0]` by `[0,2]` case returns an ordinary error on Metal with no resident-byte leak (`TestEmptyResidentMatMulSizeRefusal`). General resident dispatch-size arithmetic is unchanged; no failing resident route was confirmed within this finding.

## Review

Independent Astra consultation identified the default/Gemm, rounded/batched and runtime-allocation boundaries before implementation. A fresh sol security/compatibility review inspected the complete candidate and direct consumers, ran focused CPU and actual GPU regressions and found no confirmed remaining scoped defect or documentation/specification contradiction. It raised the existing resident arithmetic as an unproven separate concern; the concrete reported resident example was then verified to reject safely as above.

## References checked

- Local Go 1.26.9 `runtime/slice.go:102-117` and `runtime/malloc.go:213-220`: slice allocation checks include multiplication and platform-dependent `maxAlloc`; the tests execute that refusal path. [Official versioned runtime source](https://github.com/golang/go/blob/go1.26.9/src/runtime/slice.go#L102).
- [PR #674 finding](https://github.com/HazelnutParadise/insyra/pull/674#discussion_r4443593729).

## 0.4 port and specification delivery

The correction and regression file were ported to 0.4 without conflict. Every changed matrix/allocation function body is byte-identical between the two lines, while their pre-existing differences are retained. On 0.4, build, full nn tests, vet and lint pass (0 lint issues); the same targeted host/Metal acceptance/control suite, race detector and strict Python SafeTensors reference round trips pass. Strict main-spec validation passes on both lines (214 dev, 236 on 0.4).

The change was validated strictly before implementation, then archived in the same PR with the existing nn-inference capability updated. No new capability or placeholder Purpose was created. All six local implementation/delivery tasks are complete. Both PR bodies describe this correction and link the checked runtime behavior; their live CI/review state remains the external merge gate.
