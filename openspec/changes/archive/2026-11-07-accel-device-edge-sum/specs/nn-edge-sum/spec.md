## ADDED Requirements

### Requirement: EdgeSum runs on a device with the host's bits

`Tape.EdgeSum` on weights and values on one device, on a tape in `accel.PrecisionFloat32Rounded`, SHALL run on that device when the topology was moved there with `EdgeTopology.To`, and its output and both gradients SHALL stay on the device and equal the host's bits. Device operands with a topology on the host SHALL be refused with an error that names `To`, and a released topology SHALL be refused.

#### Scenario: Same bits as the host

- **WHEN** a topology, weights and values with cancelling, subnormal, overflowing, NaN, infinite and signed-zero entries run through `Tape.EdgeSum` and its backward pass on a device and on the host
- **THEN** the output, the weight gradient and the value gradient have the same bits

#### Scenario: Topology left on the host

- **WHEN** weights and values are on a device and the topology is not
- **THEN** `Tape.EdgeSum` returns an error saying to move the topology with `To`

#### Scenario: Moving to the host

- **WHEN** `EdgeTopology.To` is given the zero Device
- **THEN** it returns the topology itself, and `Release` does nothing
