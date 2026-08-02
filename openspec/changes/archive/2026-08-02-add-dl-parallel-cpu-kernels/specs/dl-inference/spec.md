## ADDED Requirements

### Requirement: Hot kernels parallelize without changing a single bit

`MatMul` (the 2-D fast path and the batched path) and `Conv` SHALL distribute
their independent output elements across CPU cores while preserving the serial
accumulation order within every output element, so that parallel and serial
execution produce bit-identical results. Inputs below a measured size
threshold SHALL take the serial path unchanged.

#### Scenario: Parallel MatMul is bit-identical to serial

- **WHEN** a MatMul large enough to cross the parallel threshold is computed
- **THEN** every output element SHALL equal the serial result exactly, not
  within a tolerance, and the existing one-op and whole-model parity suites
  SHALL pass unchanged

#### Scenario: Parallel Conv is bit-identical to serial

- **WHEN** a Conv large enough to cross the parallel threshold is computed
- **THEN** every output element SHALL equal the serial result exactly, and
  the fixed-weight CNN whole-model proof SHALL pass unchanged

#### Scenario: Small inputs avoid parallel overhead

- **WHEN** a MatMul or Conv is below the parallel size threshold
- **THEN** it SHALL execute on the serial path with no goroutine dispatch
