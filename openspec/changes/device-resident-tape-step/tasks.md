## 1. Device buffers
- [ ] 1.1 Add resident buffers to `accel`: create, upload, read back, release, charged to the memory budget, with a stub backend for tests
- [ ] 1.2 Add `PrecisionFloat32Rounded` to `accel.Precision` with `String()` and carry it through `Workload.Precision` and `Result.Precision`

## 2. Tensor residency
- [ ] 2.1 Add `Tensor.To`, `ToHost`, `Device`; refuse host views of resident tensors with an error naming `ToHost`
- [ ] 2.2 Test the round trip bit for bit (NaN payloads, infinities, signed zeros) and the no-device path

## 3. Kernels
- [ ] 3.1 MatMul on buffers in `PrecisionFloat32Rounded` (exact register), with transposed operands, equal to a CPU reference and a `math/big` oracle
- [ ] 3.2 Add with broadcast, broadcast-reduce, gradient accumulation and SGD on buffers, same oracle
- [ ] 3.3 Measure each against the all-core CPU on a shape ladder and record the numbers in `delivery-status.md`

## 4. Tape
- [ ] 4.1 `Tape.SetPrecision`/`GetPrecision`, `Clear`, `Close`; dispatch `MatMul`, `Add`, `Backward`, `SGD` by residency
- [ ] 4.2 Fall back in the open for operations without a kernel; strict GPU mode refuses; mixed devices are an error
- [ ] 4.3 Prove a resident step equals the host step bit for bit, and equals it with no device

## 5. Docs and release notes
- [ ] 5.1 `Docs/nn.md`: both modes, what differs, when to use which; English and Traditional Chinese kept in step
- [ ] 5.2 `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`, headings `### Core`/`### nn`
- [ ] 5.3 Update `delivery-status.md` (M38 progress, next change) and the skills only if a principle changed
- [ ] 5.4 `openspec validate --specs --strict`, `golangci-lint run`, `go test ./nn/... ./accel/...`, and the gated device tests on the M3
