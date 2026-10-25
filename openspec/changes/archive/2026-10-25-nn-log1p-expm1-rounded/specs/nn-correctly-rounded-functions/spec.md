## ADDED Requirements

### Requirement: Log1p and expm1 are correctly rounded on the host and the device

`log1p(x)` and `expm1(x)` SHALL return, for every float32 input, the true value rounded once to the nearest float32 with ties to even, with `log1p(±0) = ±0`, `log1p(-1) = -∞`, NaN below -1, and `expm1(-∞) = -1`. The device kernels SHALL return the same bits as the host.

#### Scenario: Every input

- **WHEN** the device and the host evaluate `log1p` and `expm1` on all 2^32 float32 bit patterns
- **THEN** every result is bit for bit equal, and a sample of results equals an independent high-precision oracle
