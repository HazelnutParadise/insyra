## Context
Insyra needs to support Intel, AMD integrated graphics, CUDA devices, and Apple GPUs while staying opt-in and multi-platform. A single backend cannot cover all required targets well enough for v1.

## Goals / Non-Goals
- Goals:
  - normalize `CUDA`, `Metal`, and `WebGPU native` into one discovery model
  - make device selection explainable and deterministic
  - allow heterogeneous multi-device planning for shardable operations
- Non-Goals:
  - making ROCm the primary AMD integrated path
  - adding vendor-specific policy branches to core `insyra`

## Decisions
- Decision: prefer `CUDA` on NVIDIA, `Metal` on Apple, otherwise `WebGPU native`
  - Rationale: this matches the best-supported primary path for each platform while keeping a portable route for Intel and AMD iGPU/APU support.
- Decision: normalize discovery output into one `Device` surface
  - Rationale: later scheduler and CLI changes need backend-independent metadata such as vendor, device type, memory class, and score.
- Decision: allow multiple devices only when the operation is shardable
  - Rationale: v1 should not promise transparent multi-device execution for non-partitionable work.

## Risks / Trade-offs
- Risk: portable backend capabilities may lag vendor-native backends.
  - Mitigation: keep capability flags explicit and use reports to explain why a device was or was not selected.
- Risk: discovery order could hide better secondary devices.
  - Mitigation: rank all discovered devices and make the chosen-device list observable.
