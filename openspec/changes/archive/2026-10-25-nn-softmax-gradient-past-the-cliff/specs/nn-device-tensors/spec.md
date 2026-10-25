## ADDED Requirements

### Requirement: Rounded cross-entropy keeps the label's gradient on host and device

In `PrecisionFloat32Rounded`, `SoftmaxCrossEntropy` SHALL give the label's gradient as the negated quotient of the exact sum of the other exponentials, rounded once, by the exact sum of all of them, rounded once, times `upstream/rows`; the host and the device SHALL return the same bits.

#### Scenario: Logits far past the cliff

- **WHEN** logits whose label leads by 20, 40, 60 and 90 are differentiated on the host and on a device
- **THEN** the label's gradient is nonzero, each row sums to zero within rounding, and the device equals the host bit for bit
