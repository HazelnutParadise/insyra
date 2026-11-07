## 1. Implementation

- [x] 1.1 A `base` word in every elementwise kernel of `resident_kernels.go` and `runElementwise` launching them in bounded spans
- [x] 1.2 `TestRecurrentStepTiming` in `nn`

## 2. Verification

- [x] 2.1 `TestElementwiseLaunchesCoverEveryElement`; the existing oracle and parity tests pass unchanged
- [ ] 2.2 The recurrent-step measurement completes on the GT 1030 at a million nodes with a batch of 16

## 3. Docs

- [x] 3.1 Both changelogs, the cross-entropy follow-up in `AGENTS.md`
- [ ] 3.2 `delivery-status.md`: the measurement
