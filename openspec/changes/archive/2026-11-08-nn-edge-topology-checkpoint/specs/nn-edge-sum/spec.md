## ADDED Requirements

### Requirement: An edge topology saves and loads with every check before it is built

`nn.SaveEdgeTopology` SHALL write a topology's node count and edge list, in edge order, as a versioned SafeTensors file, the same bytes for a topology on a device as for its host original, reading nothing from the device. `nn.LoadEdgeTopology` SHALL check the format version and every tensor's dtype and shape before reading the data, SHALL read no more data than the file holds, and SHALL check every index before returning a host topology.

#### Scenario: A round trip
- **WHEN** a topology is saved and loaded
- **THEN** the loaded topology is `Equal` to it and saves the same bytes

#### Scenario: A bad file
- **WHEN** the file is empty, truncated, followed by data, from another format or version, or names an unknown, missing or mistyped tensor, an index outside the nodes, or no nodes
- **THEN** `LoadEdgeTopology` returns an error and no topology

#### Scenario: A header that claims more data than the file holds
- **WHEN** the header describes a billion edges and the file holds a few bytes
- **THEN** the load fails after reading what the file holds, without allocating what the header claims

### Requirement: Equal compares the edges in order

`(*EdgeTopology).Equal` SHALL report whether two topologies have the same node count and the same source and target at every edge index, and SHALL NOT compare where they live. Two nil topologies SHALL be equal and a nil one SHALL equal no other.

#### Scenario: The same edges in another order
- **WHEN** two topologies hold the same edges with two of them swapped
- **THEN** `Equal` reports false

### Requirement: A resident recurrent run continues in a new process

A sparse recurrent run on a device that saves its topology, weights and state through `ToHost`, and its optimizer state, at a completed step SHALL continue in a new process, restored in the order topology, weights and state, optimizer state, to the same bits it would have reached without stopping, and SHALL reach the same bits when resumed on the host.

#### Scenario: Three steps, a restart, four steps
- **WHEN** a resident `EdgeSum`, `Add`, `Tanh` and Adam run trains three steps, saves, and a new process restores it and trains four more
- **THEN** its topology, weights, state and optimizer state equal those of seven uninterrupted steps byte for byte, and the same holds when the checkpoint resumes on the host
