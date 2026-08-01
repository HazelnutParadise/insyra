# Tasks

## 1. Implementation
- [x] 1.1 Optional `FitWeighted` on `Estimator`
- [x] 1.2 `CrossValidateWeighted` — weight validation on WLS's terms, per-fold subsetting aligned with training rows, unweighted held-out scoring
- [x] 1.3 Refusals: missing `FitWeighted`, invalid or misaligned weights

## 2. Verification
- [x] 2.1 An estimator that records what it was handed proves per-fold weight–row alignment exactly — the misalignment this change exists to prevent
- [x] 2.2 WLS end-to-end through weighted cross-validation
- [x] 2.3 Every refusal
- [x] 2.4 Existing unweighted paths untouched: the full suite still passes

## 3. Documentation
- [x] 3.1 `Docs/ml.md`: the weighted variant, the unweighted-scoring convention, the closed limitation
- [x] 3.2 `skills/insyra/` update
- [x] 3.3 Changelogs in both languages; `delivery-status.md` handoff note updated to one remaining open question
