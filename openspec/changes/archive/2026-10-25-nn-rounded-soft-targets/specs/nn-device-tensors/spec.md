## ADDED Requirements

### Requirement: The rounded cross-entropy loss survives a large margin

In `PrecisionFloat32Rounded`, a row's hard-label loss SHALL be `RN(ell - d[label])` with `ell = log1p(o)` and `o` the exact sum of the row's exponentials but the first peak's, rounded once, on the host and on the device with the same bits.

#### Scenario: A label far past the cliff

- **WHEN** logits `[40, 0]` are scored against label 0
- **THEN** the loss is `log1p(e^-40)` rounded to float32, not 0

### Requirement: Rounded soft-target cross-entropy is reproducible

In `PrecisionFloat32Rounded`, `SoftTargetCrossEntropy` SHALL compute each row's loss as the exact sum of `p*ell - p*d` rounded once and the gradient as `u = RN(q*P - p)` for every class but the peak, with the peak's gradient minus the rounded sum of the others, each scaled by `upstream/rows`; the host and the device SHALL return the same bits.

#### Scenario: Host and device agree

- **WHEN** rows with a label past the cliff, tied logits and a one-class row are scored on the host and on a device
- **THEN** the loss and every gradient are equal bit for bit

### Requirement: Rounded distillation follows the paper's gradient

In `PrecisionFloat32Rounded`, `DistillationLoss` SHALL give the gradient of a row whose teacher probabilities are all above 0 and whose centred differences are at most 1 by Barron's Eq. 9 with every step rounded once, and otherwise by `q - p`, on the host and the device with the same bits.

#### Scenario: Equal logits and a vanishing difference

- **WHEN** a student equals its teacher, and a student `[1e-20, 0, 1]` is scored against a teacher `[0, 0, 1]`
- **THEN** the first gradient is exactly +0 and the second is nonzero, equal on host and device
