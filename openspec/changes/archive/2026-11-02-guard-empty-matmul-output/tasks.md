## 1. Reproduce

- [x] 1.1 Write tests for SafeTensors-to-MatMul/Gemm output overflow, runtime allocation refusal and bounded huge-empty execution; observe the original failures.
- [x] 1.2 Cover ordinary empty/vector/batch shapes, hook exclusion, rounded transpose variants and Gemm Alpha/C semantics.

## 2. Correct and verify

- [x] 2.1 Guard output dispatch/allocation and remove work on empty matrix products without changing valid arithmetic.
- [x] 2.2 Run focused tests, full nn tests, build, vet, lint and relevant host/Metal regressions; review the final patch independently.
- [x] 2.3 Update nn docs and both changelogs, archive in the same PR and pass strict specs validation.

## 3. Deliver

- [x] 3.1 Port the correction to 0.4, repeat verification and update both PRs before the authorized all-green CI/clean-review merge gate.
