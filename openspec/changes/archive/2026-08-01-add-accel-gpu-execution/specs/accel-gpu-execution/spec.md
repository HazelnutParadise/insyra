## ADDED Requirements

### Requirement: Real device execution for an eligible column reduction
The system SHALL execute an eligible columnar reduction on a discovered acceleration device and return the computed value to the caller.

#### Scenario: Eligible column is reduced on a real device
- **WHEN** an accel session holds a discovered device with a registered execution backend and is asked to reduce an eligible numeric column
- **THEN** the runtime uploads the column to the device, dispatches a compute kernel, reads the result back, and returns the reduced value
- **AND** the returned value equals the value the CPU path computes for the same column

#### Scenario: Column is empty
- **WHEN** the column to be reduced contains no rows
- **THEN** the runtime returns the additive identity without dispatching any work to the device

#### Scenario: No execution backend is registered
- **WHEN** a device is discovered but no execution backend is registered for its backend kind
- **THEN** the runtime does not report the workload as accelerated
- **AND** it reports a fallback reason naming the missing backend

### Requirement: Execution seam carries an operation and a result
The system SHALL expose a backend execution seam that receives the operation to perform and returns the computed result, an error, and measured cost data.

#### Scenario: A backend is registered for a device kind
- **WHEN** an execution backend is registered for a backend kind
- **THEN** the runtime routes eligible workloads for devices of that kind through the registered backend
- **AND** the seam passes the requested operation and the projected column data to the backend
- **AND** the seam returns the computed result, a cancellation-aware error, and measured timings

#### Scenario: Backend execution fails
- **WHEN** a registered execution backend returns an error
- **THEN** the runtime does not report the workload as accelerated
- **AND** the error and its fallback reason are visible in the session report
- **AND** in strict GPU mode the error is returned to the caller rather than absorbed by CPU fallback

### Requirement: Reduced precision is opt-in and never silent
The system SHALL NOT reduce the precision of a caller's data unless the caller has explicitly accepted reduced precision for that operation, and SHALL report the precision every accelerated result was computed at.

#### Scenario: Caller does not accept reduced precision
- **WHEN** a 64-bit floating point column is submitted without the caller accepting single-precision execution
- **THEN** the runtime does not execute it on the device
- **AND** it reports a fallback reason identifying precision as the cause
- **AND** the CPU path produces the result

#### Scenario: Caller accepts single-precision execution
- **WHEN** a 64-bit floating point column is submitted and the caller has explicitly accepted single-precision execution
- **THEN** the runtime narrows the column for device execution and returns the computed value
- **AND** the result records that it was computed at single precision

#### Scenario: Column type cannot be represented on the device at any precision
- **WHEN** a column's type has no device representation even under an explicit precision opt-in
- **THEN** the runtime reports a fallback reason identifying the column type as ineligible

#### Scenario: Shader compiler accepts a type its backend cannot lower
- **WHEN** a shader compiler accepts a type its target backend cannot lower
- **THEN** eligibility is still decided by the runtime before submission
- **AND** the runtime does not rely on the shader compiler to reject unsupported column types

### Requirement: Observable failure reasons for device execution
The system SHALL report a distinct, stable reason for each way device execution can fail.

#### Scenario: Kernel fails to compile
- **WHEN** the backend cannot compile the compute kernel for the selected device
- **THEN** the session report carries a fallback reason identifying shader compilation as the cause

#### Scenario: Column exceeds a device buffer limit
- **WHEN** the column does not fit within the device's maximum storage buffer binding size
- **THEN** the runtime divides the work into chunks that fit
- **AND** if a single chunk still cannot fit, the session report carries a fallback reason identifying the buffer limit as the cause

#### Scenario: Result readback does not complete
- **WHEN** reading the result back from the device does not complete before the configured deadline
- **THEN** the session report carries a fallback reason identifying readback timeout as the cause

### Requirement: Software and CPU adapters are not acceleration devices
The system SHALL NOT treat a backend-reported CPU or software adapter as an acceleration device.

#### Scenario: The backend offers only a software adapter
- **WHEN** the execution backend can supply an adapter but that adapter reports itself as a CPU or software renderer
- **THEN** the runtime does not enumerate it as an acceleration device
- **AND** workloads fall back to the CPU path with a fallback reason stating that no hardware device was available

#### Scenario: A host has both a hardware device and a software adapter
- **WHEN** the backend offers both a hardware device and a software adapter
- **THEN** the runtime selects the hardware device and ignores the software adapter

### Requirement: Measured execution cost instead of estimated cost
The system SHALL report execution cost measured from real device activity, and SHALL NOT report cost values derived from fixed per-backend constants.

#### Scenario: A workload runs on a device
- **WHEN** a workload completes on a real device
- **THEN** the session report exposes measured transfer, dispatch, and readback durations for that execution

#### Scenario: No workload ran on a device
- **WHEN** no workload was executed on a device
- **THEN** the session report does not present transfer or dispatch cost figures for that execution

### Requirement: GPU dependency stays outside the core module
The system SHALL keep GPU driver dependencies out of the core `insyra` module, and SHALL make an execution backend available only to consumers that opt in.

#### Scenario: A consumer installs the library without acceleration
- **WHEN** a consumer adds the core `insyra` module to a project
- **THEN** no GPU runtime dependency is added to that project's module requirements

#### Scenario: A consumer opts into GPU execution
- **WHEN** a consumer adds the acceleration backend module and imports it for its registration side effect
- **THEN** the backend registers itself with the accel runtime
- **AND** eligible workloads are routed to it without further configuration
