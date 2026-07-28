## ADDED Requirements

### Requirement: Device-independent results
The system SHALL return the same result for an accepted request whether or not a device performed the work.

#### Scenario: No device is present
- **WHEN** a caller requests distances or nearest query points on a host where no accelerator was discovered
- **THEN** the runtime computes the result on the CPU and returns it
- **AND** the result is reported as not accelerated, with the reason naming why the device did not run

#### Scenario: The device fails part way through
- **WHEN** device execution fails, times out, exceeds a buffer limit, or its shader does not compile
- **THEN** the runtime computes the result on the CPU and returns it
- **AND** the reported reason names the device failure rather than reporting success

#### Scenario: The caller's own terms exclude every available path
- **WHEN** a request is refused because the caller did not accept reduced precision, or because no kernel accepts the column types
- **THEN** the runtime returns no result
- **AND** the reported reason names the ineligible request

#### Scenario: Strict GPU mode is in force
- **WHEN** a caller has opted into strict GPU mode and no device can run the work
- **THEN** the runtime returns an error rather than a CPU result
