## ADDED Requirements

### Requirement: The device tanh equals the host's

`accel.RoundedTanh` SHALL return, for every element of a resident float32 array, the same bits `nn.Tanh` returns on the host for that input, computed on the device in integer arithmetic only, so no shader compiler can change a result. A NaN SHALL give a NaN.

#### Scenario: Every input
- **WHEN** the device and the host evaluate `tanh` on all 2^32 float32 bit patterns
- **THEN** every result is bit for bit equal, NaN for NaN

#### Scenario: Against an independent oracle
- **WHEN** a sample of inputs covering both analytic boundaries, the inputs the host settles from its table, and random inputs runs on the device
- **THEN** every result equals the correctly rounded `tanh` from a `math/big` oracle
