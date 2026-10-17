## ADDED Requirements

### Requirement: The exported surface is the user API
The `accel` package SHALL export only what a user, `accel/knnbridge`, `nn` or the CLI calls. Backend registration, device and report injection, the CPU reference for nearest search, and test reset hooks SHALL be unexported. Their former exported names SHALL remain for one release, marked `Deprecated:`, and SHALL behave as before.

#### Scenario: The backend-author names are deprecated
- **WHEN** the package's exported declarations are read
- **THEN** `RegisterBackendExecutor`, `BackendExecutor`, `ExecuteRequest`, `ExecuteResponse`, `ExecuteColumn`, `RegisterDiscoverer`, `Discoverer`, `RegisterSDKProbe`, `SDKProbe`, `Session.RegisterDevice`, `Session.RecordReport`, `Session.PlanShardableWorkload`, `NearestExactCPU` and the four `Reset…ForTest` functions each carry a `Deprecated:` paragraph

#### Scenario: A deprecated name still works
- **WHEN** a caller uses `Session.RegisterDevice` during the deprecation release
- **THEN** the device is registered exactly as before the change

#### Scenario: Tests in another package reset the shared session without the exported hook
- **WHEN** a test outside `accel` needs the shared session rediscovered
- **THEN** it calls `internal/acceltest.ResetDefault`, which drops the session that `accel.Default` returns
