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

### Requirement: SDK probe priority over native command probes and env stubs
The system SHALL prefer SDK-backed probes (vendor libraries such as NVML for CUDA) over native host-command probes, and SHALL prefer either of those over environment-variable stubs.

#### Scenario: SDK probe and native command probe both report devices for the same backend
- **WHEN** an SDK probe registered for a backend returns one or more devices
- **THEN** the runtime uses the SDK-reported devices as the primary inventory for that backend
- **AND** the runtime does not consult native command probes or env stubs for that backend in the same discovery pass

#### Scenario: SDK probe reports the host does not have the SDK installed
- **WHEN** every SDK probe for a backend reports that the SDK is unavailable
- **THEN** the runtime falls through to the native host-command probe for that backend
- **AND** falls through to env-variable stubs only if the native probe also reports unavailable

#### Scenario: SDK probe surfaces driver-level metadata
- **WHEN** an SDK probe returns a device
- **THEN** the unified device view exposes driver version, CUDA compute capability, and PCI bus identifier when the SDK reports them
- **AND** marks the device with an `sdk` probe source so observers can distinguish SDK-discovered devices from native-command and env-stub discoveries
