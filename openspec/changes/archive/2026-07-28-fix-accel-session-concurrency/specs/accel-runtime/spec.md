## ADDED Requirements

### Requirement: Session is safe for concurrent use
The system SHALL allow any number of goroutines to share one `Session`, with every public method safe to call concurrently with every other.

#### Scenario: Concurrent executions on one session
- **WHEN** multiple goroutines execute workloads through the same session at the same time
- **THEN** each execution's result is identical to what it would have produced running alone
- **AND** the race detector reports no data race

#### Scenario: Reads during execution
- **WHEN** one goroutine is executing a workload and another reads reports, devices, plans, or cache snapshots
- **THEN** the reader observes a consistent state, never a partially applied update

#### Scenario: Concurrent sessions share one device
- **WHEN** two sessions execute on the same host device at the same time
- **THEN** device submissions are serialized rather than interleaved
