## ADDED Requirements

### Requirement: 2-D MatMul keeps each output's dot-product order
The system SHALL compute every output element of a 2-D float32 `MatMul` on the CPU as one running sum that starts at zero and adds the products `a[i][p]*b[p][j]` in increasing `p`, so that every non-NaN output is bit-identical to that per-element dot product however the loops over rows and columns are arranged.

#### Scenario: The kernel matches the per-element dot product
- **WHEN** a 2-D MatMul is computed serially or in parallel on inputs that hold signed zeros, infinities and subnormal products
- **THEN** every output that is not NaN SHALL have the same bits as the per-element dot product, and every NaN output SHALL be NaN

#### Scenario: An empty inner dimension gives zeros
- **WHEN** a 2-D MatMul multiplies an m×0 tensor by a 0×n tensor
- **THEN** the result SHALL be an m×n tensor of positive zeros
