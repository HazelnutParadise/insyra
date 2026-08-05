# accel-gpu-execution Delta: add-accel-chunked-submission

## ADDED Requirements

### Requirement: Oversized submissions are chunked, not lost
The system SHALL split a device submission whose size exceeds a measured bound into sequential bounded chunks on the same device, SHALL merge per-chunk results into an answer bit-identical to the unchunked one, and SHALL leave submissions at or below the bound byte-for-byte unchanged.

#### Scenario: A shape that timed out now completes

- **WHEN** the shape whose single submission previously died in readback-timeout runs with chunking
- **THEN** the operation completes on the device
- **AND** its results equal brute force exactly

#### Scenario: Chunked equals unchunked

- **WHEN** a shape small enough to run unchunked is forced through the chunked path
- **THEN** both paths return identical results

#### Scenario: The bound is a recorded measurement

- **WHEN** the chunk bound is inspected
- **THEN** its value traces to the recorded saturation curve with a stated margin, not to an unexplained constant

#### Scenario: Observability reports the chunks

- **WHEN** a chunked execution completes
- **THEN** the execution result reports how many chunks ran, and everything else in the result surface is unchanged
