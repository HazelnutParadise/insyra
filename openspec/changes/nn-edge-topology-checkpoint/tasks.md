## 1. Implementation

- [x] 1.1 `SaveEdgeTopology`, `LoadEdgeTopology` and `EdgeTopology.Equal` in `nn/edge_topology_io.go`

## 2. Verification

- [x] 2.1 Round trip, edge-order comparison and every bad-file case, including a header that claims more data than the file holds
- [x] 2.2 A resident topology saves its host bytes; a resident recurrent run resumes in a new process and on the host bit for bit (`INSYRA_ACCEL_GPU_TESTS=1`)

## 3. Docs

- [x] 3.1 `Docs/nn.md` restore order, both changelogs
- [x] 3.2 `delivery-status.md`: M39 done
