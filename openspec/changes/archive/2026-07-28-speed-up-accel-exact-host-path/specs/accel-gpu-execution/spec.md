## ADDED Requirements

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
