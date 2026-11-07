## Why

M36 of #379: `EdgeSum` and its gradients ran only on the host, so a recurrent step whose tensors live on a GPU had to bring them back for every edge sum. The 2026-09-26 measurement showed the exact arithmetic is as fast on the device as on all CPU cores, and that uploading on every call loses; the kernels have to read inputs already on the device.

## What Changes

- `accel.NewDeviceEdgeTopology` uploads an edge list once, grouped by target and by source; `RoundedEdgeSum`, `RoundedEdgeSumGradValues` and `RoundedEdgeSumGradWeights` sum along it on resident buffers, each output the exact sum of its products rounded once.
- `nn.EdgeTopology` gains `To`, `Device` and `Release`, like a tensor. `Tape.EdgeSum` on weights and values on a device, in `PrecisionFloat32Rounded`, needs the topology moved there and keeps the output and both gradients on the device.
- The CSR grouping moves to `internal/edgecsr`, shared by `nn` and `accel`.

## Impact

- `accel/edge_sum.go`, `accel/internal/wgpu/edge_sum.go`, `nn/edge_sum.go`, `nn/edge_sum_device.go`, `nn/autodiff_edge_sum.go`, `internal/edgecsr`; `Docs/nn.md`, `Docs/accel.md`, both changelogs, `delivery-status.md`. API: additions only.
