# Proposal: chisq-labels-follow-number-text

## Why

Two defects the adversarial review of `chisq-observed-expected` found, recorded as an `AGENTS.md` follow-up and approved by the owner on 2026-09-29:

- The chi-square tests label each category with `fmt`'s `%v`, so a float category of 1,500,000 is labelled `1.5e+06` and 0.00001 is labelled `1e-05`. Every other text output of insyra (`ToStringSlice`, `ToCSV`, CCL) writes `1500000` and `0.00001`. The labels are the row and column names of `Observed` and `Expected`, and since `chisq-observed-expected` they are also the keys `ChiSquareGoodnessOfFit`'s `p` needs. So a caller had to write `{"1.5e+06": 0.5}`, a spelling found nowhere else in the library.
- `ChiSquareIndependenceTest` reads `rowData.Data()` and then `colData.Data()`. A writer that resizes both lists together can land between the two reads, so the test pairs values from two states of the data. The two-sample tests read both samples under one `AtomicDoAll` for this reason (`TestTwoSampleTTestReadsBothSamplesAtOneMoment`).

## What Changes

- `ChiSquareIndependenceTest` reads both lists under one `insyra.AtomicDoAll`, so a caller who resizes both together never gets a mixed read.
- Left on 0.4: the label rule for float categories, which is **BREAKING** (a category outside the middle range gets a new label), together with its docs. On this line category labels keep `fmt`'s `%v`, so `1.5e+06` is still the label of 1,500,000.

## Capabilities

### New Capabilities

- `stats-chi-square`: adds "The independence test reads both lists at one moment". This line has no main spec for the capability yet, so the delta creates it.

### Modified Capabilities

None.

## Impact

- Code: `stats/chi_square.go`.
- Tests: the concurrent-resize test for the independence test in `stats/test_pair_atomicity_test.go`.
