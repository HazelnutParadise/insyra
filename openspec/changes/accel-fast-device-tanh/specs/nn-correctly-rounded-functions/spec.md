## ADDED Requirements

### Requirement: The device tanh settles most inputs in 64-bit arithmetic

The device `tanh` SHALL first evaluate a 64-bit fixed-point approximation whose error has a stated bound, SHALL round it only when it lies farther than that bound from every float32 rounding midpoint, and SHALL otherwise use the 128-bit evaluation, so that its result is unchanged: the bits `nn.Tanh` gives on the host for every float32 input.

#### Scenario: The error bound holds
- **WHEN** the host model of the approximation runs on inputs across its range, from reciprocal starts moved by 2^-18 either way, against a 600-bit oracle
- **THEN** its largest error is at most an eighth of the bound

#### Scenario: Few inputs fall back
- **WHEN** the host model runs on every positive input of its range
- **THEN** at most a thousand inputs are sent to the 128-bit evaluation

#### Scenario: Every input still equals the host
- **WHEN** the device and the host evaluate `tanh` on all 2^32 float32 bit patterns
- **THEN** every result is bit for bit equal
