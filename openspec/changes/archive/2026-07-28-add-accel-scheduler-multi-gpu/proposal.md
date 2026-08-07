# Change: Define acceleration scheduler and heterogeneous multi-GPU policy

## Why
Acceleration needs an explicit scheduler contract before implementation starts. The phase must define what counts as shardable work, how devices are weighted, how results merge, and when the runtime falls back to CPU.

## What Changes
- Add a new `accel-scheduler` capability
- Define shard planning for heterogeneous multi-device execution
- Define weighted partitioning, deterministic merge, and strict-vs-auto behavior
- Define CPU fallback boundaries for unsupported or unprofitable workloads

## Impact
- Affected specs: `accel-scheduler`
- Affected code: future planner, scheduler, runtime execution path, and CPU merge behavior
