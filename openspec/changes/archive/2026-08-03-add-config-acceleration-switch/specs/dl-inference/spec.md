## ADDED Requirements

### Requirement: Acceleration obeys the Config system, layered under the ops override

`insyra.Config` SHALL expose `SetAcceleration(enabled bool)` and
`GetAccelerationEnabled()`, default enabled, and every device path — dl's
device MatMul, the KNN bridge, accel session opening — SHALL consult it at
call time. A device SHALL run only when both the Config switch and the
`INSYRA_ACCEL_DISABLE_WGPU` environment override allow it; either alone
SHALL force the byte-identical CPU path.

#### Scenario: Config turns devices off programmatically

- **WHEN** a program calls `insyra.Config.SetAcceleration(false)` and then
  runs an above-floor dl matmul or an eligible KNN search
- **THEN** the device SHALL NOT be consulted and results SHALL be the
  existing CPU results byte-for-byte

#### Scenario: The env override wins over Config

- **WHEN** `INSYRA_ACCEL_DISABLE_WGPU=1` is set and Config acceleration is
  enabled
- **THEN** devices SHALL stay off

#### Scenario: Re-enabling restores the device path

- **WHEN** acceleration is disabled and later re-enabled via Config with no
  env override set
- **THEN** subsequent eligible operations SHALL use the device again
