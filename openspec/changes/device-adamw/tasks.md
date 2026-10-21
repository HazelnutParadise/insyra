## 1. Host
- [x] 1.1 Rounded Adam and AdamW on host tensors, equal to a `math/big` oracle for the sequence
- [x] 1.2 Host scalar helpers (`1-beta^t` via `math/big`, `lr*wd`)

## 2. Device
- [x] 2.1 Integer division and square root in WGSL, equal to the host on hardware
- [x] 2.2 Adam kernel with resident moments; `accel` public function

## 3. Tape
- [x] 3.1 Dispatch `Adam`/`AdamW` by residency; state allocated on the device and released on `Close`
- [x] 3.2 A multi-step run equal on host and device

## 4. Docs and release notes
- [x] 4.1 godoc on every exported name; `Docs/nn.md` kernel coverage
- [x] 4.2 `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`
- [x] 4.3 `delivery-status.md`; `openspec validate --specs --strict`, `golangci-lint run`, tests, gated device tests
