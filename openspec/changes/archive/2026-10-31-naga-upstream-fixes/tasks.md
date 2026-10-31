# Tasks

- [x] `go.mod` requires gogpu/naga v0.20.4; the `go` directive is unchanged and `go build ./...` passes.
- [x] `ieee_decode` already uses `countLeadingZeros(fx) - 8u` on this line; the upgrade makes it right on Vulkan.
- [x] The `select()` ban is gone from `exact_sum.go`, `exact_sum_test.go` and `ENG.md`.
- [x] `Docs/nn.md` drops the `-race` note.
- [x] `TestWGSLBuiltinsMatchTheHost` passes on an Apple M3 over Metal and fails there with naga v0.19.0.
- [x] `INSYRA_ACCEL_GPU_TESTS=1 go test ./accel/... ./nn/...` passes on the M3. `-race` ran on `dev`'s port of this change, with the same gogpu/wgpu v0.34.5.
- [x] An entry in the `### `accel`` section above `## Unreleased` in both CHANGELOGs.
- [x] The device tests pass on an NVIDIA GT 1030 over Vulkan (`TestWGSLBuiltinsMatchTheHost` and the subnormal division, square root, exp and log tests, 2026-10-31).
