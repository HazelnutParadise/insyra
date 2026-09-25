# nn-edge-sum Specification

## Purpose
A sparse edge-sum operation for graphs given as edge lists: each node sums its weighted incoming edges without a dense N×N matrix, with a reverse rule for the edge weights and the node values and a fixed summation order a device kernel can be held to.

## Requirements

### Requirement: An edge topology is validated once and never changes

`NewEdgeTopology(nodes, sources, targets)` SHALL refuse a non-positive `nodes`, `sources` and `targets` of different lengths, and any index outside `[0, nodes)`, naming the offending edge. It SHALL copy the slices it is given, so changing them afterwards does not change the topology. An empty edge list SHALL be allowed.

#### Scenario: An index out of range
- **WHEN** `NewEdgeTopology(3, []int{0, 1}, []int{1, 3})` is called
- **THEN** it returns an error naming edge 1 and the index 3

#### Scenario: The caller changes its slice afterwards
- **WHEN** a topology is built from `sources` and the caller then overwrites `sources[0]`
- **THEN** `EdgeSum` on that topology gives the same result as before the overwrite

### Requirement: Each node sums its weighted incoming edges

`EdgeSum(topology, weights, values)` SHALL return `out[..., t] = Σ weights[e]·values[..., sources[e]]` over every edge `e` with `targets[e] = t`, and zero for a node with no incoming edge. `weights` SHALL be float32 of shape `[E]`, `values` float32 of shape `[N]` or `[B, N]`, and the output SHALL have the shape of `values`. Anything else SHALL be an error. Work and memory SHALL be O(E + size of `values`), independent of N².

#### Scenario: The three-edge example from #379
- **WHEN** sources `[0, 1, 2]`, targets `[1, 2, 0]`, weights `[0.5, -1, 0.25]` and values `[0.1, 0.2, 0.3]`
- **THEN** the output is `[0.25·0.3, 0.5·0.1, -1·0.2]` in float32

#### Scenario: Repeated targets and negative weights
- **WHEN** several edges share a target and some weights are negative
- **THEN** the output agrees with a dense float64 product within float32 tolerance

#### Scenario: A large sparse graph
- **WHEN** a graph has a million nodes and a handful of edges
- **THEN** `EdgeSum` completes without allocating anything proportional to N²

### Requirement: The summation order is fixed

Each output SHALL be computed by starting from zero and adding the edges of its target in ascending edge index, with every product rounded to float32 before it is added. The result SHALL be bit-identical to that loop on every platform.

#### Scenario: The same result as the reference loop
- **WHEN** `EdgeSum` runs on a graph with repeated targets
- **THEN** every output is bit-identical to the ascending-edge loop with explicitly rounded products

### Requirement: The operation has a reverse rule on the tape

`Tape.EdgeSum` SHALL record the operation so that the gradient of `values[..., s]` is the sum over edges with source `s`, in ascending edge index, of `weights[e]·upstream[..., targets[e]]`, and the gradient of `weights[e]` is the sum over the batch, in ascending batch index, of `upstream[b, targets[e]]·values[b, sources[e]]`, every product rounded to float32 before it is added.

#### Scenario: Gradients against finite differences
- **WHEN** `EdgeSum` feeds `Tanh` and a loss, and the gradients of `weights` and `values` are compared with central finite differences
- **THEN** they agree within the tolerance the perturbation allows

#### Scenario: Gradients against the reference loops
- **WHEN** the tape's gradients are compared with the ascending-order loops above
- **THEN** they are bit-identical

### Requirement: Large edge sums use every core without changing a bit

`EdgeSum` and the two gradients of `Tape.EdgeSum` SHALL divide their output elements across cores when the work exceeds `nn`'s parallel threshold, with each output element computed by exactly one worker in the order the summation-order requirement fixes. The result SHALL be bit-identical whatever the number of workers.

#### Scenario: One worker and every worker agree
- **WHEN** the forward pass and both gradients run on a graph large enough to use every core, once with one worker and once with every worker
- **THEN** every output and every gradient is bit-identical between the two

#### Scenario: A small graph stays on one core
- **WHEN** the work is at or below the parallel threshold
- **THEN** the operation uses a single worker

### Requirement: A device edge sum is earned by measurement and matches the CPU bit for bit

No production device kernel for `EdgeSum` SHALL exist until a recorded measurement shows the device faster than the all-core CPU at the measured sizes, with every upload and readback counted, and shows a kernel variant whose results are bit-identical to the CPU's contracted order. The measurement and its verdict SHALL be recorded in `delivery-status.md`.

#### Scenario: The device is measured before a kernel is proposed
- **WHEN** a device path for `EdgeSum` is proposed
- **THEN** `delivery-status.md` already records device and all-core CPU times per size and the bit-for-bit comparison of each kernel variant against the CPU order

#### Scenario: The device cannot match the CPU order
- **WHEN** no kernel variant is bit-identical to the unfused CPU order
- **THEN** no device path is wired, and the choice of order goes to the owner
