## 1. Implementation
- [x] 1.1 Validate the change with `openspec validate speed-up-accel-projection --strict`
- [x] 1.2 Write the tests first: named numeric types still project, mixed int/float still becomes float64, an unconvertible value still errors with its index, float32 narrowing matches today, and uint64 above MaxInt64 wraps the same way
- [x] 1.3 Record the before figures from `BenchmarkProjectValues` with `-benchmem`
- [x] 1.4 Replace `reflect` dispatch in `toFloat64` and `toInt64` with concrete type switches, falling back to the existing reflect path for named types
- [x] 1.5 Replace `inferDataType`'s predicate chain with one type switch per element, keeping the reflect fallback for named types
- [x] 1.6 Re-measure; if a full pass over the values still shows up materially, fold `buildValidityBitmap` into the conversion loop and re-measure again
- [x] 1.7 Record the after figures in `design.md`, and update the cost tables in `add-accel-gpu-execution/design.md` and `Docs/accel.md`
- [x] 1.8 Run `go test ./...`, `go test -race ./accel/...`, and the GPU test with `INSYRA_ACCEL_GPU_TESTS=1`
- [x] 1.9 Update `delivery-plan.md` and the `## Follow-ups` entry in `AGENTS.md`
