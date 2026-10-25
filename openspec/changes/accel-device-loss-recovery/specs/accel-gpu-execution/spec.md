## ADDED Requirements

### Requirement: A lost device is reported and replaced
The system SHALL report a lost GPU device as `ErrDeviceLost`, SHALL refuse every buffer and binding made on it with that error, and SHALL open a new device for the next call that needs one.

#### Scenario: A dispatch is stopped by the operating system
- **WHEN** a dispatch runs long enough for the device to be lost
- **THEN** the call that meets the loss returns an error for which `errors.Is(err, ErrDeviceLost)` is true
- **AND** reading a buffer made before the loss returns the same error

#### Scenario: Work after the loss
- **WHEN** a caller creates a new buffer and binds the same kernel after the loss
- **THEN** the kernel runs on a new device and returns the correct result

### Requirement: Built-in operations survive a lost device
The system SHALL return a correct result from a built-in operation whose call meets a device loss, by computing it on the CPU, and SHALL use the new device for later calls.

#### Scenario: Nearest-row search meets a loss
- **WHEN** the device is lost during an exact nearest-row search
- **THEN** the search returns the CPU result with the fallback reason `device-lost`
