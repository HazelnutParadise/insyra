## ADDED Requirements

### Requirement: Large edge sums use every core without changing a bit

`EdgeSum` and the two gradients of `Tape.EdgeSum` SHALL divide their output elements across cores when the work exceeds `nn`'s parallel threshold, with each output element computed by exactly one worker in the order the summation-order requirement fixes. The result SHALL be bit-identical whatever the number of workers.

#### Scenario: One worker and every worker agree
- **WHEN** the forward pass and both gradients run on a graph large enough to use every core, once with one worker and once with every worker
- **THEN** every output and every gradient is bit-identical between the two

#### Scenario: A small graph stays on one core
- **WHEN** the work is at or below the parallel threshold
- **THEN** the operation uses a single worker
