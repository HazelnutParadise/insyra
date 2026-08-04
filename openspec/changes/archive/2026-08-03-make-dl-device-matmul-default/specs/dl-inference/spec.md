## REMOVED Requirements

### Requirement: Large matmuls may run on a device, invisibly and exactly

**Reason**: The opt-in bridge design this requirement described is replaced
wholesale by default-on wiring. Its cycle argument only held for the bridge
package itself; the exactness, floor, fallback, and refusal obligations
survive unchanged in the replacing requirement.

## ADDED Requirements

### Requirement: Large matmuls run on a device by default, invisibly and exactly

`dl` SHALL run 2-D f32 matmuls at or above the measured MAC floor on a
device by default, wiring the device implementation at package init through
`accel`'s exported surface — no opt-in import. Setting
`INSYRA_ACCEL_DISABLE_WGPU=1` or calling `RegisterDeviceMatMul(nil)` SHALL
restore the pure CPU path. Device results SHALL remain asserted bit-equal to
the CPU path on hardware, and any device absence, error, or parity failure
SHALL fall back to the CPU path observably rather than change any answer.
Under the `race` build tag the device path SHALL NOT be wired.

#### Scenario: Default on, results exact

- **WHEN** a program imports `dl` with no further configuration, a device is
  present, and a 2-D matmul at or above the measured floor executes
- **THEN** the result SHALL be bit-equal to the CPU path's result, asserted
  by a hardware test with exact equality

#### Scenario: The switch restores the CPU path

- **WHEN** `INSYRA_ACCEL_DISABLE_WGPU=1` is set or the hook is cleared with
  `RegisterDeviceMatMul(nil)`
- **THEN** every matmul SHALL take the pure CPU path and produce the
  existing CPU results byte-for-byte

#### Scenario: Device trouble is a performance event

- **WHEN** the device is missing, errors, or the platform fails the parity
  assertion
- **THEN** the matmul SHALL return the CPU result, the fallback SHALL be
  observable, and only strict GPU mode SHALL fail instead

#### Scenario: Below the floor and batched shapes stay on the CPU

- **WHEN** a matmul is batched or below the measured MAC floor
- **THEN** the device SHALL NOT be consulted, because measurement refused
  those shapes
