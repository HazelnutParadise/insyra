# Proposal: add-accel-device-selection

## Why

A library user on a multi-GPU host has no hard way to control which devices insyra may touch: `Config.PreferredDevices` is a soft ordering hint consumed during primary-device selection, and no environment variable masks devices at all. Every established accelerator ecosystem separates hard selection from soft preference — `CUDA_VISIBLE_DEVICES`, TensorFlow's `set_visible_devices`, ONNX Runtime's ordered providers — and multi-device dispatch (the next change) needs a defined answer to "which devices" before it can exist.

## What Changes

- `Config.Devices` — a hard allowlist of device IDs (or zero-based discovery indices); empty means all. A session never touches a device outside it.
- `INSYRA_ACCEL_DEVICES` — a process-wide hard mask applied at the discovery boundary, mirroring `CUDA_VISIBLE_DEVICES` semantics: masked devices are invisible downstream, to every session.
- Interaction semantics defined and tested: eligible set = env mask ∩ `Config.Devices`; strict modes error on an empty eligible set; automatic modes fall back to CPU with a new `FallbackReason` naming the cause; `PreferredDevices` keeps its soft-ordering role within the eligible set.
- Docs and both changelogs.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `accel-runtime`: sessions SHALL honor hard device bounds from configuration and environment, with defined strict/automatic behavior when the bounds leave nothing eligible.

## Impact

- **Code**: discovery filtering and session configuration in `accel/`; a new `FallbackReason` value. API is additive — one new `Config` field, one new env var.
- **Behavior**: default behavior unchanged (empty allowlist, no env var = today).
- **Docs**: `Docs/accel.md`, `CHANGELOG.md` / `CHANGELOG_TW.md`.
- **Dependencies**: none.
