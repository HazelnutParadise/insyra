## ADDED Requirements

### Requirement: Tanh runs where its operand lives

`Tape.Tanh` on a resident float32 tensor, on a tape in `PrecisionFloat32Rounded` or `PrecisionFloat32`, SHALL compute its output and its gradient on the device and leave both there, with the bits the same tape gives on host tensors.

#### Scenario: A recurrent-style step stays resident
- **WHEN** a tape with resident parameters runs `MatMul`, `Add`, `Tanh`, `MatMul`, `Backward` and `SGD`
- **THEN** every output, gradient and updated parameter equals the host run of the same tape bit for bit

#### Scenario: Gradient edge cases
- **WHEN** the input holds `±0`, values whose `tanh` rounds to `±1`, infinities, NaN and subnormal products
- **THEN** the device gradient equals the host's bit for bit, NaN for NaN
