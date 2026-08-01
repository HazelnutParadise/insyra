# Change: Define acceleration backend discovery

## Why
The accel runtime needs a single discovery contract that covers NVIDIA, Apple, Intel, and AMD integrated/shared-memory devices. Without a normalized discovery spec, backend-specific implementations would diverge in naming, eligibility, and default selection behavior.

## What Changes
- Add a new `accel-device-discovery` capability
- Define backend probing for `CUDA`, `Metal`, and `WebGPU native`
- Define normalized device metadata, scoring, and default selection policy
- Define multi-device enumeration and heterogeneous eligibility for shardable workloads

## Impact
- Affected specs: `accel-device-discovery`
- Affected code: future backend adapters and runtime session initialization
