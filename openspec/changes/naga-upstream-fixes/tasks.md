# Tasks

- [x] `go.mod` requires gogpu/naga v0.20.4; the `go` directive is unchanged and `go build ./...` passes.
- [x] `ieee_decode` uses `countLeadingZeros(fx) - 8u`.
- [x] The `select()` ban is gone from `exact_sum.go`, `exact_sum_test.go` and `ENG.md`.
- [x] `Docs/accel.md` drops the `countLeadingZeros` warning; `Docs/nn.md` and `delivery-status.md` drop the `-race` notes.
- [x] `TestWGSLBuiltinsMatchTheHost` passes on an Apple M3 over Metal and fails there with naga v0.19.0.
- [x] `INSYRA_ACCEL_GPU_TESTS=1 go test -race ./accel/... ./nn/...` passes on the M3.
- [x] Entries under `## Unreleased` / `### `accel`` in both CHANGELOGs.
- [ ] The device tests pass on an NVIDIA GT 1030 over Vulkan.
