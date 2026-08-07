## ADDED Requirements

### Requirement: Large matmuls may run on a device, invisibly and exactly

`dl` SHALL expose a device-matmul hook that is nil by default, consulted only
for 2-D f32 matmuls at or above a measured MAC floor, and filled only by
blank-importing an opt-in bridge package. With the hook nil the CPU path SHALL
be byte-for-byte what it was before this change. With the hook active, device
results SHALL be asserted bit-equal to the CPU path on hardware, and any
device absence, error, or parity failure SHALL fall back to the CPU path
observably rather than change any answer.

#### Scenario: No bridge, no change

- **WHEN** a program uses `dl` without blank-importing the bridge
- **THEN** no device is discovered, no accel code is linked beyond the hook
  declaration, and every result is the existing CPU result

#### Scenario: Bridge active, results exact

- **WHEN** the bridge is blank-imported, a device is present, and a 2-D matmul
  at or above the measured floor executes
- **THEN** the result SHALL be bit-equal to the CPU path's result, asserted by
  a hardware test with exact equality

#### Scenario: Device trouble is a performance event

- **WHEN** the bridge is active but the device is missing, errors, or the
  platform fails the parity assertion
- **THEN** the matmul SHALL return the CPU result, the fallback SHALL be
  observable, and only strict GPU mode SHALL fail instead

#### Scenario: Below the floor and batched shapes stay on the CPU

- **WHEN** a matmul is batched or below the measured MAC floor
- **THEN** the hook SHALL NOT be consulted, because measurement refused those
  shapes
