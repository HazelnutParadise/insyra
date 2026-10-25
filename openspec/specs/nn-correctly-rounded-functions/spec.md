# nn-correctly-rounded-functions Specification

## Purpose
Elementwise float32 functions whose results are correctly rounded — the true value rounded once to the nearest float32 with ties to even — so the CPU and any device agree by definition, and the reproducible evaluation of their gradients as fixed sequences of correctly rounded basic operations.
## Requirements
### Requirement: Tanh is correctly rounded

`nn.Tanh` SHALL return, for every float32 input, the true value of `tanh(x)` rounded to the nearest float32 with ties to even. It SHALL return `x` for `±0`, `±1` for `±∞`, and a NaN for a NaN. The result SHALL NOT depend on the platform.

#### Scenario: Every input
- **WHEN** `nn.Tanh` is evaluated on all 2^32 float32 bit patterns
- **THEN** every result between the two analytic ranges equals the correctly rounded value from an independent high-precision oracle, and every result inside them equals its proven value

#### Scenario: Inputs near a rounding boundary
- **WHEN** the float64 approximation for an input lies within `tanhSettleSteps` float64 steps of the midpoint between two float32 values
- **THEN** the result comes from the table of those inputs, whose entries equal the oracle, not from the approximation

#### Scenario: The approximation is the same on every platform
- **WHEN** the fast path runs on any platform
- **THEN** it computes the same float64 bits as on the platform where the exhaustive run measured its error: no product is fused into the operation after it, and a hash of the bits over every input between the analytic ranges is checked on every platform CI runs

### Requirement: The tanh gradient is reproducible

`Tape.Tanh`'s gradient SHALL be `RN(upstream · RN(1 − RN(y·y)))`, where `y` is the forward output and every operation is rounded to float32 and none is fused with another.

#### Scenario: Against the step-by-step reference
- **WHEN** the gradient is compared with each step computed in float64 and rounded to float32
- **THEN** every element is bit-identical, on every platform

### Requirement: Exp and log are correctly rounded
`nn` SHALL provide float32 `exp` and `log` whose result is the exact value rounded once to nearest-even, on every platform, computed with IEEE `+ - * /` only. Special inputs SHALL follow IEEE 754: `exp(-inf)` is `+0`, `exp(+inf)` is `+inf`, `log(0)` is `-inf`, `log` of a negative is NaN.

#### Scenario: Exhaustive agreement
- **WHEN** each function is compared with its oracle over every float32 input
- **THEN** every result has the oracle's bits

#### Scenario: The device agrees
- **WHEN** the device versions run on every input on hardware
- **THEN** every result has the host's bits

### Requirement: Log1p and expm1 are correctly rounded on the host and the device

`log1p(x)` and `expm1(x)` SHALL return, for every float32 input, the true value rounded once to the nearest float32 with ties to even, with `log1p(±0) = ±0`, `log1p(-1) = -∞`, NaN below -1, and `expm1(-∞) = -1`. The device kernels SHALL return the same bits as the host.

#### Scenario: Every input

- **WHEN** the device and the host evaluate `log1p` and `expm1` on all 2^32 float32 bit patterns
- **THEN** every result is bit for bit equal, and a sample of results equals an independent high-precision oracle

