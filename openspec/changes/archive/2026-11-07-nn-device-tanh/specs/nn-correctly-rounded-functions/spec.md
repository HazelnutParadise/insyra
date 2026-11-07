## MODIFIED Requirements

### Requirement: The tanh gradient is reproducible

`Tape.Tanh`'s gradient SHALL be `RN(upstream · RN(1 − RN(y·y)))`, where `y` is the forward output and every operation is rounded to float32 and none is fused with another. In `PrecisionFloat32Rounded` and `PrecisionFloat32` a result that is exactly zero SHALL be `+0` and a nonzero result that rounds to zero SHALL keep its sign; the default tape SHALL keep IEEE's sign of zero.

#### Scenario: Against the step-by-step reference
- **WHEN** the gradient is compared with each step computed in float64 and rounded to float32
- **THEN** every element is bit-identical, on every platform

#### Scenario: A zero gradient in the rounded mode
- **WHEN** `tanh` rounds to `±1` and the upstream gradient is negative
- **THEN** the rounded mode's gradient is `+0` and the default tape's is `-0`

#### Scenario: A gradient that underflows
- **WHEN** the exact gradient is negative and smaller than half the smallest subnormal
- **THEN** both modes give `-0`
