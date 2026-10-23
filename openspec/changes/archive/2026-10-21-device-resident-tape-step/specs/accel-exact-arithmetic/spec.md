## ADDED Requirements

### Requirement: Float32Rounded fixes one answer for every device
`accel.PrecisionFloat32Rounded` SHALL mean that each float32 result is the exact value of its operation rounded once to nearest-even, so that every device, every driver and the CPU fallback return the same bits. It SHALL be stricter than `PrecisionFloat32` and looser than `PrecisionExact` in what it asks of the caller, and it SHALL never be the default. The mode SHALL appear in `Workload.Precision` and `Result.Precision`, and a device that cannot meet it SHALL be reported as a fallback, not used.

#### Scenario: Two devices agree
- **WHEN** one operation runs in `PrecisionFloat32Rounded` on two different devices and on the CPU
- **THEN** all results have the same bits

#### Scenario: The default is untouched
- **WHEN** no precision is requested
- **THEN** behaviour is that of `PrecisionExact`, as before
