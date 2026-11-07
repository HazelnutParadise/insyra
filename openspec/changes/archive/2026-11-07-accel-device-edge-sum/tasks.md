## 1. Kernels

- [x] 1.1 `edgeSumRoundedKernel` and `edgeSumWeightsRoundedKernel` on the exact register, registered for warmup
- [x] 1.2 `accel.DeviceEdgeTopology` and the three `RoundedEdgeSum*` functions, validating device, lengths and release

## 2. nn

- [x] 2.1 `EdgeTopology.To`, `Device`, `Release`; `Tape.EdgeSum` on resident operands with resident gradients
- [x] 2.2 Bit parity with the host on random, cancelling, subnormal, overflowing, NaN/Inf, signed-zero and edgeless cases on the M3
- [x] 2.3 Refusals: host topology with device tensors, released topology, tape not in `PrecisionFloat32Rounded`

## 3. Measurement and docs

- [x] 3.1 Kernel time ladder against the all-core CPU, recorded in design.md and delivery-status.md
- [x] 3.2 `Docs/nn.md`, `Docs/accel.md`, both changelogs
