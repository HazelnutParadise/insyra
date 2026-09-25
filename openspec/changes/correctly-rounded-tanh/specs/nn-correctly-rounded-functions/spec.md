## Purpose

Elementwise float32 functions whose results are correctly rounded — the true value rounded once to the nearest float32 with ties to even — so the CPU and any device agree by definition, and the reproducible evaluation of their gradients as fixed sequences of correctly rounded basic operations.

## ADDED Requirements

### Requirement: Tanh is correctly rounded

`nn.Tanh` SHALL return, for every float32 input, the true value of `tanh(x)` rounded to the nearest float32 with ties to even. It SHALL return `x` for `±0`, `±1` for `±∞`, and a NaN for a NaN. The result SHALL NOT depend on the platform.

#### Scenario: Every input
- **WHEN** `nn.Tanh` is evaluated on all 2^32 float32 bit patterns
- **THEN** every result equals the correctly rounded value from an independent high-precision oracle

#### Scenario: Inputs near a rounding boundary
- **WHEN** the float64 `tanh` of an input lies close to the midpoint between two float32 values
- **THEN** the result is decided by a high-precision evaluation, not by the float64 value

### Requirement: The tanh gradient is reproducible

`Tape.Tanh`'s gradient SHALL be `RN(upstream · RN(1 − RN(y·y)))`, where `y` is the forward output and every operation is rounded to float32 and none is fused with another.

#### Scenario: Against the step-by-step reference
- **WHEN** the gradient is compared with each step computed in float64 and rounded to float32
- **THEN** every element is bit-identical, on every platform
