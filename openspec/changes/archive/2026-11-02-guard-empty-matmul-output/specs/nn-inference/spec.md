## ADDED Requirements

### Requirement: Matrix products validate output size before work
MatMul and Gemm SHALL reject output element-count overflow or runtime allocation-size refusal with an error before computation. MatMul SHALL validate before device dispatch and return empty or zero-inner products without row loops or batch-offset allocation, preserving scalar/vector/batch shapes and existing float32 bits. Gemm SHALL retain Alpha and C broadcasting semantics. Valid empty SafeTensors SHALL remain supported.

#### Scenario: Empty inputs imply an unrepresentable output
- **WHEN** zero-payload tensors such as [MaxInt,0] and [0,2] are loaded and passed to MatMul or Gemm, including rounded or transposed variants
- **THEN** multiplication returns an error instead of panicking, and the device hook is not called

#### Scenario: Runtime allocation limit is exceeded
- **WHEN** an output count fits int but the runtime refuses its float32 slice allocation
- **THEN** multiplication returns an allocation error without changing the inputs

#### Scenario: Huge empty rows or batches
- **WHEN** a matrix product has an empty output or zero inner dimension, including vector promotion and huge empty batch dimensions
- **THEN** it returns the correctly shaped result without iterating empty rows or allocating per-batch offsets

#### Scenario: Gemm preserves zero-inner arithmetic
- **WHEN** Gemm has a zero inner dimension with nonempty output and a supplied Alpha or bias
- **THEN** its signed zeros, NaNs and C broadcasting validation match the existing computation

## MODIFIED Requirements

### Requirement: 2-D MatMul keeps each output's dot-product order
The system SHALL compute every output element of a 2-D float32 `MatMul` on the CPU as one running sum that starts at zero and adds the products `a[i][p]*b[p][j]` in increasing `p`, so that every non-NaN output is bit-identical to that per-element dot product however the loops over rows and columns are arranged.

#### Scenario: The kernel matches the per-element dot product
- **WHEN** a 2-D MatMul is computed serially or in parallel on inputs that hold signed zeros, infinities and subnormal products
- **THEN** every output that is not NaN SHALL have the same bits as the per-element dot product, and every NaN output SHALL be NaN

#### Scenario: An empty inner dimension gives zeros
- **WHEN** a 2-D MatMul multiplies an m×0 tensor by a 0×n tensor and the output allocation is representable
- **THEN** the result SHALL be an m×n tensor of positive zeros
