## ADDED Requirements

### Requirement: Multi-backend discovery support
The system SHALL support acceleration backend discovery across `CUDA`, `Metal`, and `WebGPU native`.

#### Scenario: Session starts on a supported host
- **WHEN** an accel session starts on a host with one or more supported acceleration runtimes
- **THEN** the session probes the available backends and enumerates eligible devices from them

### Requirement: Normalized device metadata
The system SHALL normalize discovered devices into a common metadata model.

#### Scenario: Different backends expose different native properties
- **WHEN** devices are discovered from different backends
- **THEN** the runtime exposes a unified device view including backend, vendor, device type, shared-memory vs discrete-memory class, budget information, and scheduler score

### Requirement: Default backend selection policy
The system SHALL apply a deterministic default backend selection policy.

#### Scenario: User runs with automatic backend selection
- **WHEN** the accel mode is automatic
- **THEN** the runtime prefers `CUDA` for NVIDIA devices, `Metal` for Apple devices, and otherwise `WebGPU native`
- **AND** it may include additional devices only if the operation is shardable and the selection policy justifies them

### Requirement: Heterogeneous multi-device eligibility
The system SHALL allow heterogeneous multi-device eligibility for shardable operations.

#### Scenario: Large shardable workload is planned
- **WHEN** a shardable columnar workload is large enough to benefit from multiple devices
- **THEN** the runtime may include more than one device even if they come from different supported backends
