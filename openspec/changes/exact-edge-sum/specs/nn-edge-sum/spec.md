## ADDED Requirements

### Requirement: Each output is the correctly rounded exact sum

Each output of `EdgeSum` SHALL be the exact sum of its products `weights[e]·values[..., sources[e]]`, rounded once to the nearest float32 with ties to even, including subnormal results and overflow to infinity. The result SHALL NOT depend on the order of the edges, the order of the batch, or the number of workers. A NaN operand, a product of zero and infinity, or infinite products of both signs SHALL make the output NaN; otherwise an infinite product SHALL make the output that infinity. An output whose exact sum is zero, including one with no incoming edge, SHALL be `+0`.

#### Scenario: Against an exact oracle
- **WHEN** `EdgeSum` runs on random graphs with repeated targets and negative weights
- **THEN** every output equals the `math/big` exact sum of its products rounded to float32

#### Scenario: Cancellation
- **WHEN** a node's products include `x`, `-x` and a value far smaller than `x`
- **THEN** the output is that small value, not zero

#### Scenario: Ties, subnormals and overflow
- **WHEN** a node's exact sum lies exactly half-way between two float32 values, or below the smallest normal float32, or beyond the largest
- **THEN** the output is the tie rounded to even, the correctly rounded subnormal, or the infinity round-to-nearest gives

#### Scenario: Edge order does not matter
- **WHEN** the same edges are given to `NewEdgeTopology` in a different order, with their weights permuted to match
- **THEN** every output is bit-identical

### Requirement: The reverse rule is the exact sum too

`Tape.EdgeSum` SHALL record the operation so that the gradient of `values[..., s]` is the sum over edges with source `s` of `weights[e]·upstream[..., targets[e]]`, and the gradient of `weights[e]` is the sum over the batch of `upstream[b, targets[e]]·values[b, sources[e]]`, each gradient the correctly rounded exact sum of its products under the same rules as the forward pass.

#### Scenario: Gradients against finite differences
- **WHEN** `EdgeSum` feeds `Tanh` and a loss, and the gradients of `weights` and `values` are compared with central finite differences
- **THEN** they agree within the tolerance the perturbation allows

#### Scenario: Gradients against an exact oracle
- **WHEN** the tape's gradients are compared with the `math/big` exact sums of their products rounded to float32
- **THEN** they are bit-identical

## MODIFIED Requirements

### Requirement: Each node sums its weighted incoming edges

`EdgeSum(topology, weights, values)` SHALL return `out[..., t] = Σ weights[e]·values[..., sources[e]]` over every edge `e` with `targets[e] = t`, and zero for a node with no incoming edge. `weights` SHALL be float32 of shape `[E]`, `values` float32 of shape `[N]` or `[B, N]`, and the output SHALL have the shape of `values`. Anything else SHALL be an error. Work and memory SHALL be O(E + size of `values`), independent of N².

#### Scenario: The three-edge example from #379
- **WHEN** sources `[0, 1, 2]`, targets `[1, 2, 0]`, weights `[0.5, -1, 0.25]` and values `[0.1, 0.2, 0.3]`
- **THEN** the output is `[0.25·0.3, 0.5·0.1, -1·0.2]`, each product rounded once to float32

#### Scenario: Repeated targets and negative weights
- **WHEN** several edges share a target and some weights are negative
- **THEN** the output equals the exact sum of its products rounded once to float32

#### Scenario: A large sparse graph
- **WHEN** a graph has a million nodes and a handful of edges
- **THEN** `EdgeSum` completes without allocating anything proportional to N²

### Requirement: Large edge sums use every core without changing a bit

`EdgeSum` and the two gradients of `Tape.EdgeSum` SHALL divide their output elements across cores when the work exceeds `nn`'s parallel threshold. The result SHALL be bit-identical whatever the number of workers.

#### Scenario: One worker and every worker agree
- **WHEN** the forward pass and both gradients run on a graph large enough to use every core, once with one worker and once with every worker
- **THEN** every output and every gradient is bit-identical between the two

#### Scenario: A small graph stays on one core
- **WHEN** the work is at or below the parallel threshold
- **THEN** the operation uses a single worker

### Requirement: A device edge sum is earned by measurement and matches the CPU bit for bit

No production device kernel for `EdgeSum` SHALL exist until a recorded measurement shows the device faster than the all-core CPU at the measured sizes, with every upload and readback counted, and shows a kernel whose results are bit-identical to the CPU's. The measurement and its verdict SHALL be recorded in `delivery-status.md`.

#### Scenario: The device is measured before a kernel is proposed
- **WHEN** a device path for `EdgeSum` is proposed
- **THEN** `delivery-status.md` already records device and all-core CPU times per size and the bit-for-bit comparison of the kernel against the CPU

#### Scenario: The device cannot match the CPU order
- **WHEN** a kernel is not bit-identical to the CPU's result
- **THEN** no device path is wired

## REMOVED Requirements

### Requirement: The summation order is fixed

**Reason**: Replaced by "Each output is the correctly rounded exact sum", which a device can reproduce under WGSL's guarantees and which does not depend on how the work is split. The fixed order depended on WGSL keeping a floating-point loop's order and rounding, which it does not promise.

**Migration**: None. `EdgeSum` is unreleased. Code that reproduced the ascending order to compare results should compare against the exact sum rounded once.

### Requirement: The operation has a reverse rule on the tape

**Reason**: Replaced by "The reverse rule is the exact sum too". Its gradients were defined by the removed ascending order.

**Migration**: None. `Tape.EdgeSum` is unreleased; its gradients are now the correctly rounded exact sums.
