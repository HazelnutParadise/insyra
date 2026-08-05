# accel-gpu-execution Specification

## Purpose
TBD - created by archiving change add-accel-distance-kernel. Update Purpose after archive.
## Requirements
### Requirement: A device result is bit-identical to its CPU reference
The system SHALL ship a CPU reference implementation for every device operation, and SHALL verify on the running platform that the device result matches it bit for bit.

#### Scenario: The kernel runs on a host with a device
- **WHEN** the squared-distance kernel executes on a device
- **THEN** every returned value is bit-identical to the CPU reference computed over the same inputs

#### Scenario: A platform cannot reach parity
- **WHEN** a platform's device result differs from the CPU reference
- **THEN** the difference is observable rather than silent

### Requirement: Device-independent results
The system SHALL return the same result for an accepted request whether or not a device performed the work.

#### Scenario: No device is present
- **WHEN** a caller requests exact nearest query points on a host where no accelerator was discovered
- **THEN** the runtime computes the result on the CPU and returns it
- **AND** the result is reported as not accelerated, with the reason naming why the device did not run

#### Scenario: The device fails part way through
- **WHEN** device execution fails, times out, exceeds a buffer limit, or its shader does not compile
- **THEN** the runtime computes the result on the CPU and returns it
- **AND** the reported reason names the device failure rather than reporting success

#### Scenario: The request itself has no answer to give
- **WHEN** a request is refused because no kernel and no reference can read the column types
- **THEN** the runtime returns no result
- **AND** the reported reason names the ineligible request

#### Scenario: Strict GPU mode is in force
- **WHEN** a caller has opted into strict GPU mode and no device can run the work
- **THEN** the runtime returns an error rather than a CPU result

### Requirement: Exact nearest query points
The system SHALL report the nearest query points per row with the same result a pure `float64` computation would produce, whether or not a device took part.

#### Scenario: The nearest query points are requested
- **WHEN** a caller asks for the `m` nearest query points for every row
- **THEN** the runtime returns `m` query indices and `m` `float64` squared distances per row, nearest first
- **AND** the indices and distances equal those of a `float64` computation over every query point

#### Scenario: A device narrows the candidates
- **WHEN** a device is available and the workload is eligible
- **THEN** the device ranks the query points in single precision and returns a shortlist per row
- **AND** the returned distances are recomputed in `float64` before any of them is chosen

#### Scenario: The shortlist boundary cannot be trusted
- **WHEN** the distance of the best rejected candidate lies within the single-precision error bound of the last accepted one
- **THEN** that row is recomputed against every query point in `float64`
- **AND** the error bound grows with the number of dimensions rather than being a fixed constant

#### Scenario: Two query points are equally close
- **WHEN** a row is exactly equidistant from more than one query point in `float64`
- **THEN** the lowest of those query indices is reported first

#### Scenario: How much rechecking happened is reported
- **WHEN** the operation completes
- **THEN** the number of rows recomputed against every query point is reported

#### Scenario: More neighbours are requested than exist
- **WHEN** a caller asks for more nearest query points than there are query points
- **THEN** the request is rejected before any work is scheduled

### Requirement: The host side uses the machine it is on
The system SHALL spread host-side nearest-neighbour work across the available cores when there is enough of it to be worth splitting.

#### Scenario: A large workload runs without a device
- **WHEN** exact nearest query points are computed and no device takes part
- **THEN** the work is split across the available cores
- **AND** the result is identical to computing it on one core

#### Scenario: A device shortlist is verified
- **WHEN** a device returns a shortlist and the host recomputes it in `float64`
- **THEN** that verification is split across the available cores
- **AND** the count of rows recomputed in full is the same as it would be on one core

#### Scenario: The workload is too small to split
- **WHEN** the work is below the threshold where splitting pays for itself
- **THEN** it runs on one core

### Requirement: The device is chosen by work per row
The system SHALL decide whether to use a device from how much work each row carries, not from the query count alone.

#### Scenario: Few query points but many dimensions
- **WHEN** each row must be compared against few query points that each carry many dimensions
- **AND** the resulting work per row exceeds the measured threshold
- **THEN** the device is used

#### Scenario: Many query points but few dimensions
- **WHEN** each row must be compared against many query points that each carry few dimensions
- **AND** the resulting work per row falls below the measured threshold
- **THEN** the device is declined and the reason names the unprofitable workload

### Requirement: GPU execution reaches every consumer of the core module
The system SHALL make GPU execution available to every consumer of the core module without a second install step or a registration import.

#### Scenario: A consumer installs the library
- **WHEN** a consumer adds the core `insyra` module to a project
- **THEN** GPU execution is available through the accel runtime without installing a further module
- **AND** without importing a package for its registration side effect

#### Scenario: A consumer never uses acceleration
- **WHEN** a consumer imports only non-accel packages
- **THEN** the GPU implementation is not compiled into their program
- **AND** the GPU dependency's minimum Go version does not apply to their build

#### Scenario: A third party supplies its own backend
- **WHEN** a third party registers an execution backend for a backend kind
- **THEN** the runtime routes eligible workloads for that kind to the registered backend

#### Scenario: A consumer installs through allpkgs
- **WHEN** a consumer installs Insyra through the `allpkgs` convenience package
- **THEN** the GPU backend is registered without any further import
- **AND** no device is probed until an accel session opens

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

### Requirement: Only profitable operations are offered
The system SHALL NOT offer a device operation that measurement shows is slower than the host performing the same work with every core available.

#### Scenario: An operation is measured to lose
- **WHEN** an accelerated operation is measured against a host using all its cores and does not win
- **THEN** the operation is removed rather than left available
- **AND** the measurement is recorded so the decision can be revisited on other hardware

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
