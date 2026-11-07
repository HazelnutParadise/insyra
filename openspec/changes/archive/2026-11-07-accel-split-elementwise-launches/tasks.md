## 1. Implementation

- [x] 1.1 A `base` word in every elementwise kernel of `resident_kernels.go` and `runElementwise` launching them in bounded spans
- [x] 1.2 `TestRecurrentStepTiming` in `nn`

## 2. Verification

- [x] 2.1 `TestElementwiseLaunchesCoverEveryElement`; the existing oracle and parity tests pass unchanged
- [x] 2.2 The recurrent-step measurement loses no device on the GT 1030; the row of a million nodes with a batch of 16 needs more than the card's 2 GB and is refused with `buffer-too-large`

## 3. Docs

- [x] 3.1 Both changelogs, the cross-entropy follow-up in `AGENTS.md`
- [x] 3.2 `delivery-status.md`: the measurement
