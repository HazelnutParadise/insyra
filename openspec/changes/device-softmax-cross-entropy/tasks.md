## 1. Host
- [ ] 1.1 Rounded loss and gradient on host tensors, equal to an oracle for the sequence
## 2. Device
- [ ] 2.1 int32 label buffers and `Tensor.To` for int64
- [ ] 2.2 Loss and gradient kernels; equal to the host on hardware
## 3. Tape
- [ ] 3.1 Dispatch by residency; label bound check
- [ ] 3.2 A classifier step from `MatMul` to the optimizer equal on host and device
## 4. Docs and release notes
- [ ] 4.1 godoc; `Docs/nn.md`; both changelogs; `delivery-status.md`; validation, lint, tests
