## ADDED Requirements

### Requirement: The fused cross-entropy gradient survives a large margin

The host `SoftmaxCrossEntropy` backward SHALL give the label's gradient as the negated sum of the other classes' probabilities rather than as the label's probability minus one, so that it is nonzero whenever the other classes' gradients are and each row sums to zero within float32 rounding.

#### Scenario: Logits far apart

- **WHEN** a row of logits `[m, 0]` with label 0 is differentiated for m of 20, 30, 40 and 60
- **THEN** the label's gradient equals -e^-m/(1+e^-m) to within 1e-6 relative, and is not 0
