# Proposal: ci-hygiene

## Why

Three review findings, RP-3, RP-4 and RP-5 (#277, #278, #279), plus a gap found while fixing them:

- **Formatting is not checked.** `gofmt -l .` lists 24 files. Six were in the review: two of those have been formatted since, four new ones came from this review's own batches, and 16 are in `stats/internal/fa`, which the review's count left out. The list keeps growing because golangci-lint v2 only checks formatting when `.golangci.yml` enables a formatter, and it does not.
- **The test job misses races and does not vet.** All three OS legs run `go test -v ./...` and nothing else. A full `go test -race ./...` passes today and takes 97 s on one Mac (stats 71 s, clustering 35 s, nn 31 s), so nothing stands in the way of running it in CI.
- **Workflows run with the repository's default token permissions**, and deploy-docs persists checkout credentials it does not use, since `peaceiris/actions-gh-pages` pushes with its own `github_token`. Deleting the `persist-credentials: true` line, as the review suggested, would change nothing, because `true` is the checkout default. It has to say `false`.
- **Nothing on `0.4` runs in CI.** Every workflow triggers on `main` and `dev` only, while every API-review batch lands on `0.4`. No batch on this line has run the tests on Windows or macOS, the linter, or govulncheck in CI. They were run locally, on one Mac.

## What Changes

- `.golangci.yml` enables the `gofmt` formatter, so the lint job fails on a file gofmt would change. All 24 files are formatted, `stats/internal/fa` included: its changes are comment indentation only, and exempting it would leave a hole in the rule for no benefit.
- `test.yml` runs `go vet ./...` on every OS. The ubuntu leg runs the suite with `-race`, and the macOS leg writes the total coverage to the job summary. The two stay on separate legs: together, atomic coverage counters and the race detector slowed `stats/internal/clustering` past the 10-minute test timeout, while each alone finishes in under 30 seconds on a Mac. The matrix sets `fail-fast: false`, so one leg failing does not cancel the others.
- Every workflow declares `permissions: contents: read`, except deploy-docs, which keeps `contents: write` and sets `persist-credentials: false`.
- Every workflow that runs on `dev` also runs on `0.4`.

## Capabilities

### New Capabilities

- `ci-workflows`: what CI checks, on which branches, and with what permissions. `test-suite-integrity` and `verification-integrity` already cover whether tests assert and whether reference comparisons run; this covers the workflows themselves.

### Modified Capabilities

(none)

## Impact

- `.golangci.yml`, the seven workflow files, 24 reformatted Go files (whitespace, comment indentation and import order only), `AGENTS.md` (the lint command now covers formatting), `api-review.md`, `delivery-status.md`.
- The first run on `0.4` found three failures that no Mac could have shown. Two tests asserted POSIX permission bits, which Windows does not have; the assertion now runs everywhere else. `MID('abc', 2, 10^300)` gave a different answer on amd64, fixed separately in `ccl-portable-integer-arguments`. A transient module-proxy error on windows also cancelled the ubuntu leg, which is why the matrix no longer fails fast.
- Nothing a library or CLI user sees changes here, so there is no changelog entry. The CCL fix has its own.
- deploy-docs runs only on `main`, so its credential change is first exercised at the next release.
- The parity and reference workflows take 30 to 45 minutes each. Running them on `0.4` costs nothing on a public repository.
