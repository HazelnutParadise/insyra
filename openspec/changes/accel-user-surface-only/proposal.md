# Proposal: accel-user-surface-only

## Why

`accel` exports its backend-author API and its test hooks next to the user API ([#262](https://github.com/HazelnutParadise/insyra/issues/262), api-review AC-1). An inventory on 2026-10-18 found no caller outside `accel` for any of these names:

- `RegisterBackendExecutor`, `BackendExecutor`, `ExecuteRequest`, `ExecuteResponse`, `ExecuteColumn`;
- `RegisterDiscoverer`, `Discoverer`, `RegisterSDKProbe`, `SDKProbe`;
- `Session.RegisterDevice`, `Session.RecordReport`, `Session.PlanShardableWorkload`;
- `NearestExactCPU`.

No backend lives outside the module, and a registered executor cannot add an operation anyway.

The four `Reset…ForTest` hooks are used by tests only. One of them is used by a test in `nn`.

The exposure also breaks the accel contract that `Report.Accelerated` says where the work ran. Measured on `0.4` at 8f3399dd, an outside program called `NewSession().RegisterDevice(fake)` and then `RecordReport(Report{Accelerated: true})`. Both returned nil, and `Report().Accelerated` was `true` with nothing run.

The owner chose on 2026-10-18 to make these names internal, rather than move them to a sub-package or only label them.

## What Changes

- **Deprecated**: every name listed above, plus `ResetBackendExecutorsForTest`, `ResetDefaultForTest`, `ResetDiscoverersForTest` and `ResetSDKProbesForTest`.
  - The code now runs on unexported equivalents.
  - The exported names stay for one release as wrappers or type aliases that behave exactly as before, following the one-name rule of #211.
  - An `AGENTS.md` follow-up records their removal.
- `nn`'s test resets the shared session through the module-internal `internal/acceltest`. `accel`'s own tests call the unexported functions.
- The user surface is unchanged. It is `Default`, `Open`, `NewSession`, `Config`, `Devices`, `Report`, `Reports`, `LastReport`, `Discover`, `Close`, `ProjectDataList`, `ProjectDataTable`, `ExecuteNearestExact`, `WorkloadEstimate`, `ExactNearestResult`, `Dataset`, `Buffer`, `PlanShardable`, `CacheSnapshot` and `DeviceMatMul`.

## Capabilities

### Modified Capabilities

- `accel-runtime`: which exported names are user API.

## Impact

- Code: `accel/*.go`, `internal/acceltest`, `nn/device_matmul_test.go`.
- Docs: `Docs/accel.md`, both CHANGELOGs, `api-review.md`, `delivery-status.md` and the `AGENTS.md` follow-up.
