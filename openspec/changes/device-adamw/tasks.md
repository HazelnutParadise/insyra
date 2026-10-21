## 1. Host
- [ ] 1.1 Rounded Adam and AdamW on host tensors, equal to a `math/big` oracle for the sequence
- [ ] 1.2 Host scalar helpers (`1-beta^t` via `math/big`, `lr*wd`)

## 2. Device
- [ ] 2.1 Integer division and square root in WGSL, equal to the host on hardware
- [ ] 2.2 Adam kernel with resident moments; `accel` public function

## 3. Tape
- [ ] 3.1 Dispatch `Adam`/`AdamW` by residency; state allocated on the device and released on `Close`
- [ ] 3.2 A multi-step run equal on host and device

## 4. Docs and release notes
- [ ] 4.1 godoc on every exported name; `Docs/nn.md` kernel coverage
- [ ] 4.2 `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`
- [ ] 4.3 `delivery-status.md`; `openspec validate --specs --strict`, `golangci-lint run`, tests, gated device tests
