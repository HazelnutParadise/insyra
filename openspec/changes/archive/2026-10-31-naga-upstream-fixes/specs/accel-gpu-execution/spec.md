## ADDED Requirements

### Requirement: Bit-counting and select built-ins agree with the host
The system SHALL compile device kernels with a WGSL compiler under which `countLeadingZeros`, `countTrailingZeros` and a scalar `select()` used as an operand of another expression give the host's result on every backend, so Insyra's kernels and a caller's kernel MAY use them.

#### Scenario: Bit counts on the device
- **WHEN** a kernel computes `countLeadingZeros` and `countTrailingZeros` of 0, 1, 0x80000000 and 0xffffffff
- **THEN** the results are 32, 31, 0 and 0, and 32, 0, 31 and 0, as `math/bits` gives on the host

#### Scenario: A select used as an operand
- **WHEN** a kernel computes `(v >> 24u) | select(0u, 0xffffff00u, (v & 0x80000000u) != 0u)` for v = 0x12345678
- **THEN** the result is 0x12, not the value of the select alone
