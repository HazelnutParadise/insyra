## ADDED Requirements

### Requirement: A lost device is reported as lost
The system SHALL report a lost GPU device as `ErrDeviceLost` with the fallback reason `device-lost`, and SHALL refuse every buffer and binding made on the lost device with that error.

#### Scenario: A dispatch is stopped by the operating system
- **WHEN** a dispatch runs long enough for the device to be lost
- **THEN** the call that meets the loss returns an error for which `errors.Is(err, ErrDeviceLost)` is true
- **AND** reading a buffer or submitting a binding made before the loss returns the same error

### Requirement: The next call tries a new device
The system SHALL release every object it made on a lost device and SHALL try to open a new device on the next call that needs one, reporting no device when the driver refuses.

#### Scenario: The driver gives a new device
- **WHEN** a caller creates a new buffer and binds the same kernel after the loss, and a new device opens
- **THEN** the kernel runs on the new device and returns the correct result

#### Scenario: The driver refuses a new device
- **WHEN** the driver refuses to create a device after the loss
- **THEN** creating a resident buffer fails with the fallback reason `no-accelerator`

### Requirement: Built-in operations survive a lost device
The system SHALL return a correct result from a built-in operation whose call meets a device loss, by computing it on the CPU.

#### Scenario: Nearest-row search meets a loss
- **WHEN** the device is lost during an exact nearest-row search
- **THEN** the search returns the CPU result and reports that it was not accelerated
