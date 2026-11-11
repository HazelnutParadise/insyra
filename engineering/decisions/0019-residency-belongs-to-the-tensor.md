# 0019 — Residency belongs to the tensor, and the tape has a rounded mode

**Status:** accepted
**Date:** 2026-10-21

## Context
Issue 494 asked to keep tape tensors resident on the accelerator during training. A device cannot reproduce the default tape's float64 `AdamW`, because WebGPU has no `f64`, and record 0014 requires a device float32 result to be the exact value rounded once.

## Decision
Training on the device is M38's tensor residency widened to the backward pass and the optimizer. Residency belongs to the tensor (`To`, `ToHost`, `Device`, `Release`), as in PyTorch and JAX, so one tape differentiates across devices. The tape has two modes: the default, unchanged, and `accel.PrecisionFloat32Rounded`, where each result is the exact value rounded once to float32, the same bits on the host, on any GPU and with no GPU, and `AdamW` is defined entirely in float32. The owner agreed on 2026-10-20.

## Options rejected
- A second tape bridged to the first. Tensors that carry their device let one tape cover both.
- A tolerance-based mode. None is planned; any tolerance is discussed with the owner first.

## Consequences
- Nothing reads back by itself: the caller moves a result to the host with `ToHost`.
- `accel.PrecisionFloat32` is also accepted by `Tape.SetPrecision` (M40, 2026-10-25). It means every built-in operation as in `PrecisionFloat32Rounded`, plus caller kernels whose result depends on the device, so a step that is not bit-identical across devices is never presented as one.
- The exact device `MatMul` is bound by the integer arithmetic of the exact register, about 3 G multiply-accumulates per second on the M3 (2026-10-21, best of five): 18 to 19 ms at `[4096,96]x[96,128]`, against 39 to 44 ms for the exact CPU reference and 6.6 to 8 ms for the default all-core float32 CPU. Tiling through workgroup memory did not change it. A faster kernel needs a cheaper exact scheme per product, which is research, not tuning.
- One step of a 1024×96→128→96 classifier on the M3, best of three: default host 21 to 24 ms, rounded host 84 to 106 ms, rounded device 37 to 39 ms. Whether a resident recurrent step pays on a GPU stronger than the GT 1030 is still open (`/engineering/handoff.md`).
