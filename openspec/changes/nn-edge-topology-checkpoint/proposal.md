## Why

M39 of #379: a recurrent network whose fixed sparse topology, per-edge weights, state and Adam moments live on the GPU must continue in a new process with the same next step bit for bit, with every check done before anything is built. #376 already saves and restores resident Adam moments, and weights and state save through `ToHost` and `SaveSafeTensors`. The topology cannot be saved, and its edge list cannot be read back out; since the weights are one value per edge in edge order, restoring them against a topology whose edges are in another order is wrong with no error.

## What Changes

- `nn.SaveEdgeTopology(w, g)` writes the topology as a SafeTensors file: I64 `sources` and `targets` in edge order, an I64 scalar `nodes`, and `insyra.edge_topology.version` in the metadata. A topology on a device keeps its edge list on the host, so it writes the same bytes and reads nothing back.
- `nn.LoadEdgeTopology(r)` checks the version and the three tensors' dtypes and shapes before reading their data, reads no more than the file holds, range-checks every index before converting it, and rebuilds through `NewEdgeTopology`. It returns a host topology.
- `(*EdgeTopology).Equal(other)` reports the same node count and the same edges in the same order, wherever each copy lives.
- The API was reviewed with Fable before the owner approved it.

## Impact

- `nn/edge_topology_io.go` and tests, including a subprocess restart of a resident `EdgeSum` + `Tanh` + Adam run; `Docs/nn.md`, both changelogs, `delivery-status.md`. API: three additions, on `dev` and `0.4`.
