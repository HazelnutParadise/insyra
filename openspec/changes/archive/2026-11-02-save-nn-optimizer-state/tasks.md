## 1. Tests first

- [x] 1.1 Write subprocess restart and invalid-import atomicity tests, then observe failure against the unimplemented API.
- [x] 1.2 Write repeated-Clear, gradient-policy, freezing and real-device lifecycle tests; observe new behavior fail before implementation.

## 2. Implementation

- [x] 2.1 Implement named binary export/import with complete validation and staged resident state replacement.
- [x] 2.2 Implement ClearOptions without changing Clear's signature and skip frozen parameters in every built-in optimizer.
- [x] 2.3 Reproduce and fix SafeTensors' zero-dimension count bug so scalar descriptors and empty parameter state round-trip.

## 3. Delivery

- [x] 3.1 Document API, wire format and a resume example in Docs/nn.md; add English and Taiwan Traditional-Chinese Unreleased entries.
- [x] 3.2 Run CPU and available real-device acceptance checks, go build ./..., go test ./nn/..., go vet ./... and golangci-lint run; record evidence and limitations.
- [x] 3.3 Verify specification coverage, archive this change in the same PR, replace the new capability Purpose and pass openspec validate --specs --strict.
- [x] 3.4 Port the addition to 0.4 and repeat required checks; open one reviewed API-example PR per line.

## 4. PR review and authorized merge

- [ ] 4.1 Reproduce and fix retained graph-input ownership and bounded optimizer checkpoint reading; keep all existing acceptance checks green.
- [ ] 4.2 Port both corrections, update docs/specs and rerun build/test/vet/lint on both lines.
- [ ] 4.3 Review the final patch and document fixes and verification in both PRs before the CI/review merge gate.

The owner has authorized merging each PR once its CI is all green and review has no outstanding finding. PR state records the result of that external gate.
