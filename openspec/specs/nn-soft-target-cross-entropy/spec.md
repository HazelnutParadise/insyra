# nn-soft-target-cross-entropy Specification

## Purpose
Cross-entropy in `nn` against a probability row per sample, for label smoothing and mixup, and against a teacher's logits, for distillation, with gradients that stay accurate where the textbook softmax form rounds them to zero.
## Requirements
### Requirement: Cross-entropy against a probability row

`Tape.SoftTargetCrossEntropy(logits, targets)` SHALL return the mean over rows of the negated sum of each target probability times the log-softmax of the logits, skipping classes whose target is 0, and SHALL refuse a target row that is not finite, is negative or does not sum to 1 within 1e-5, naming the row. The gradient of the logits SHALL sum to zero within float64 rounding and the targets SHALL receive none.

#### Scenario: A logit leads far past the precision cliff

- **WHEN** logits `[40, 0]` are scored against targets `[1, 0]` and differentiated
- **THEN** the loss is log1p(e^-40), the gradient of the second logit is e^-40/(1+e^-40) and the first is its negative, none of them 0

### Requirement: Distillation against a teacher's logits

`Tape.DistillationLoss(student, teacher)` SHALL return the mean over rows of the cross-entropy of the student's logits against `softmax(teacher)`, with the teacher a constant that receives no gradient, and SHALL return a gradient of exactly 0 when the two sets of logits are equal.

#### Scenario: Equal logits and a wide teacher range

- **WHEN** a student equal to the teacher is differentiated, and a student against the teacher `[1000, 0, 0]`
- **THEN** the first gradient is exactly 0 and the second loss and gradient are finite

