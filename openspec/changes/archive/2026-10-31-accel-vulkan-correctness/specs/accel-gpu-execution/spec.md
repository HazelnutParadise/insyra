## ADDED Requirements

### Requirement: A write to a resident buffer is complete before later work uses it
The system SHALL make a write to a resident buffer (an upload, `Write`, `WriteBits` or `Zero`) land before any later kernel or readback reads the buffer, on every backend.

#### Scenario: Zero then read
- **WHEN** a buffer holding nonzero words is zeroed and then read back
- **THEN** every word read is zero, in every run

### Requirement: Discovery keeps the backend that can run work
The system SHALL keep, when one card is discovered through two backends, the entry whose backend has an executor, and SHALL never prefer a backend without an executor over one that has it.

#### Scenario: An NVIDIA card seen through NVML and WebGPU
- **WHEN** discovery finds the same NVIDIA card as a CUDA device and as a WebGPU device, and only WebGPU has an executor
- **THEN** the session keeps the WebGPU device and accelerated operations run on it
