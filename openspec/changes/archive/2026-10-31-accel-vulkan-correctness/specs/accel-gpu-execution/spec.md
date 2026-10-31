## ADDED Requirements

### Requirement: Rounded arithmetic does not depend on a compiler built-in
The system SHALL compute correctly rounded division, square root, exp and log on the device from integer operations whose WGSL built-ins are identical on every backend, so that a subnormal input gives the host's result on Metal, Vulkan and DirectX 12.

#### Scenario: A subnormal division on Vulkan
- **WHEN** the device divides the float32 with bits 0x1 by the one with bits 0x2
- **THEN** the result is 0.5, as on the host

### Requirement: A write to a resident buffer is complete before later work uses it
The system SHALL make a write to a resident buffer (an upload, `Write`, `WriteBits` or `Zero`) land before any later kernel or readback reads the buffer, on every backend.

#### Scenario: Zero then read
- **WHEN** a buffer holding nonzero words is zeroed and then read back
- **THEN** every word read is zero, in every run
