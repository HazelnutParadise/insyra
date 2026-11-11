# Handoff

Unfinished work that has no ticket. Delete a section when its work is done, and the file when it is empty.

## M38: does a resident recurrent step beat the all-core CPU on any GPU?
- **Asked:** issue #379 and record 0019. `nn` tensors carry the device they live on, and a recurrent step's speed on the device is measured against the all-core CPU before more operations move to the device.
- **Done and checked:** residency, `MatMul`, `Add`, `SoftmaxCrossEntropy`, SGD, Adam and AdamW (released in v0.3.5); `EdgeSum`, `Tanh` and the topology checkpoint (unreleased: `accel-device-edge-sum`, `nn-device-tanh`, `nn-edge-topology-checkpoint`). `TestRecurrentStepTiming` times `tanh(EdgeSum(W, s) + input)` unrolled four times, the reverse pass and SGD. On an idle Windows host (i7-13700, 24 threads; GeForce GT 1030, Vulkan 1.4.312; load 0–3%, 2026-11-07) the device is 8 to 14 times slower than the rounded tape on the host at every size: host against device, 10,000 nodes, in-degree 10, batch 1: 4.1 ms against 31.6 ms; 100,000 × 10 × 16: 237 ms against 3.33 s; 100,000 × 100 × 16: 1.89 s against 18.1 s. 1,000,000 × 10 × 16 does not fit the card's 2 GB and is refused with `buffer-too-large`.
- **Tried and failed:** the first GT 1030 run lost the device on the largest row, because each elementwise kernel ran its whole array in one dispatch. `accel-split-elementwise-launches` splits them into bounded launches, and the rerun lost no device. An M3 run under a load average near 48 hinted that the device wins from 100,000 nodes with in-degree 100; it is not quoted.
- **Next:** run `TestRecurrentStepTiming` on an idle M3, or on a GPU stronger than the GT 1030, from a shell that sees the real adapter. If the device wins at some size, record the numbers in a decision record and propose the next operation to move. If it does not, no further operation moves to the device. The M36 `EdgeSum` kernel-time ladder in the archived `accel-device-edge-sum` design was taken on a loaded M3; re-measure it on the same idle host before quoting its speed-ups.
- **Sources:** [#379](https://github.com/HazelnutParadise/insyra/issues/379), [accel-split-elementwise-launches](/openspec/changes/archive/2026-11-07-accel-split-elementwise-launches/proposal.md), [nn-edge-topology-checkpoint](/openspec/changes/archive/2026-11-08-nn-edge-topology-checkpoint/proposal.md).

## Kernel warmup on a cold shader cache
- **Asked:** `accel-kernel-warmup` (#624) compiles the 28 built-in kernels in the background when the first resident buffer is created, so the first training step does not wait for them.
- **Done and checked:** measured on `dev`, with and without warmup, warm-cache first steps were equal on the M3 (12 runs each) and the GT 1030 (10 runs each). On this line only the build, lint and warmup tests ran, on the M3. Caller kernels are not warmed.
- **Tried and failed:** nothing yet.
- **Next:** clear the GT 1030 driver's shader cache and time the first step with and without warmup. A shorter first step with warmup confirms the change; an equal one means the driver cache was doing the work.
- **Sources:** [accel-kernel-warmup](/openspec/changes/archive/2026-11-01-accel-kernel-warmup/proposal.md).
