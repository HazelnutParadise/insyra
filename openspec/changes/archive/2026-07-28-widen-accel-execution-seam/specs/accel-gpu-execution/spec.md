## ADDED Requirements

### Requirement: Execution seam carries an operation and a result
The system SHALL expose a backend execution seam that receives the operation and every column it applies to, and returns the computed results, an error, and measured cost data.

#### Scenario: A backend is registered for a device kind
- **WHEN** an execution backend is registered for a backend kind
- **THEN** the runtime routes eligible workloads for devices of that kind through the registered backend
- **AND** the seam passes the requested operation and all of the dataset's projected columns in one request
- **AND** the seam returns a result per column, a cancellation-aware error, and measured timings for the submission

#### Scenario: A dataset with several columns is executed
- **WHEN** a dataset carrying more than one eligible column is executed
- **THEN** the backend receives one request naming every column
- **AND** the runtime does not submit one request per column

#### Scenario: Backend execution fails
- **WHEN** a registered execution backend returns an error
- **THEN** the runtime does not report the workload as accelerated
- **AND** the error and its fallback reason are visible in the session report
- **AND** in strict GPU mode the error is returned to the caller rather than absorbed by CPU fallback
