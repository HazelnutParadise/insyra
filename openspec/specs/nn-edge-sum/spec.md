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
