# Proposal: chisq-labels-follow-number-text

## Why

Two defects the adversarial review of `chisq-observed-expected` found, recorded as an `AGENTS.md` follow-up and approved by the owner on 2026-09-29:

- The chi-square tests label each category with `fmt`'s `%v`, so a float category of 1,500,000 is labelled `1.5e+06` and 0.00001 is labelled `1e-05`. Every other text output of insyra (`ToStringSlice`, `ToCSV`, CCL) writes `1500000` and `0.00001`. The labels are the row and column names of `Observed` and `Expected`, and since `chisq-observed-expected` they are also the keys `ChiSquareGoodnessOfFit`'s `p` needs. So a caller had to write `{"1.5e+06": 0.5}`, a spelling found nowhere else in the library.
- `ChiSquareIndependenceTest` reads `rowData.Data()` and then `colData.Data()`. A writer that resizes both lists together can land between the two reads, so the test pairs values from two states of the data. The two-sample tests read both samples under one `AtomicDoAll` for this reason (`TestTwoSampleTTestReadsBothSamplesAtOneMoment`).

## What Changes

- **BREAKING** A category's label is the value's text under the library's rule, `internal/utils.ValueText`, which `ToStringSlice` uses, with surrounding spaces removed. Floats from 0.000001 up to 1e21 are plain decimals (`1500000`, `0.00001`). Strings, integers, `nil` (`<nil>`) and floats between 0.0001 and one million keep the label they had. A float category outside that middle range gets a new label, both as a row or column name of `Observed` and `Expected` and as the key `p` needs; the old spelling becomes an unknown key and is refused.
- `ChiSquareIndependenceTest` reads both lists under one `insyra.AtomicDoAll`, so a caller who resizes both together never gets a mixed read.
- The `AGENTS.md` follow-up is resolved and deleted. `Docs/stats.md`, `Docs/cli-dsl.md`, both CHANGELOGs, `api-review.md` (a note on ST-6) and `delivery-status.md` are updated.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-chi-square`: the label rule of the goodness-of-fit probabilities, and the independence test's reads of its two lists.

## Impact

- Code: `stats/chi_square.go`.
- Tests: labels of large, small and ordinary float categories in both tests and as `p` keys; the concurrent-resize test for the independence test, run first against the old code.
- Docs: `Docs/stats.md`, `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`, `AGENTS.md`.
- No new dependencies.
