## 1. Device buffers
- [x] 1.1 Add resident buffers to `accel`: create, upload, read back, release, charged to the memory budget, with a stub backend for tests
- [x] 1.2 Add `PrecisionFloat32Rounded` to `accel.Precision` with `String()` and carry it through `Workload.Precision` and `Result.Precision`

## 2. Tensor residency
- [x] 2.1 Add `Tensor.To`, `ToHost`, `Device`; refuse host views of resident tensors with an error naming `ToHost`
- [x] 2.2 Test the round trip bit for bit (NaN payloads, infinities, signed zeros) and the no-device path

## 3. Kernels
- [x] 3.1 MatMul on buffers in `PrecisionFloat32Rounded` (exact register), with transposed operands, equal to a CPU reference and a `math/big` oracle
- [x] 3.2 Add with broadcast, broadcast-reduce, gradient accumulation and SGD on buffers, same oracle
- [x] 3.3 Measure each against the all-core CPU on a shape ladder and record the numbers in `delivery-status.md`

## 4. Tape
- [x] 4.1 `Tape.SetPrecision`/`GetPrecision`, `Clear`, `Close`; dispatch `MatMul`, `Add`, `Backward`, `SGD` by residency
- [x] 4.2 Operations without a kernel refuse a device tensor with an error naming `ToHost` (automatic host fallback arrives with the kernels that need it); mixed devices are an error
- [x] 4.3 Prove a resident step equals the host step bit for bit, and equals it with no device

## 5. Docs and release notes
- [x] 5.1 `Docs/nn.md`: both modes, what differs, when to use which; English and Traditional Chinese kept in step
- [x] 5.2 godoc on every exported name this change adds (`Tensor.To`, `ToHost`, `Device`, `Tape.Clear`, `Close`, `SetPrecision`, `GetPrecision`, `accel.PrecisionFloat32Rounded`): what it does, when to use it, what differs from the default, and an `Example` where the call sequence is not obvious
- [x] 5.3 `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`, headings `### Core`/`### nn`
- [x] 5.4 Update `delivery-status.md` (M38 progress, next change) and the skills only if a principle changed
- [ ] 5.5 `openspec validate --specs --strict`, `golangci-lint run`, `go test ./nn/... ./accel/...`, and the gated device tests on the M3
