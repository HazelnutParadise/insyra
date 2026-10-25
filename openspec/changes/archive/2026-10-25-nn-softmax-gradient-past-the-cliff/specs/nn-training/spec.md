## ADDED Requirements

### Requirement: The Softmax backward keeps the peak's gradient

The `Softmax` backward SHALL compute the gradient with the class of largest probability as pivot, so that the gradient of that probability is nonzero whenever the other classes' gradients are and the gradients of a row sum to zero within float32 rounding.

#### Scenario: Logits far apart

- **WHEN** a softmax of logits `[m, 0]` is differentiated with upstream `[1, 0]` for m of 10, 20, 30 and 40
- **THEN** the first gradient equals e^-m/(1+e^-m)^2 to within 1e-6 relative, and is not 0
