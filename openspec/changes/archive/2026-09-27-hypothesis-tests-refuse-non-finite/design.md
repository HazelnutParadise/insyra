# Design: hypothesis-tests-refuse-non-finite

## Context

`stats/numericinput.go` already holds the check the rest of the package uses. `numericValues(raw, label)` converts each cell with `insyra.ToFloat64Safe`, refuses a cell that does not convert or converts to `NaN` or `±Inf`, and names the label and the one-based row in the error. `asDataList` turns a nil interface, a typed-nil `*DataList` or a foreign `IDataList` implementation into a concrete list, so the caller's own empty or length check reports the problem. `testSeries` and `testSeriesPair` in `stats/testinput.go` combine the two for the parametric tests.

The nine functions this change covers each have a private loop over `ToFloat64Safe` that checks only `ok`. Three of them read their lists in parallel goroutines, one per group. The comments in `anova.go` record why: running the reads serially made the actor handshakes dominate wall time at more than two groups, while `TwoWayANOVA` measured the opposite and stays serial.

## Goals / Non-Goals

**Goals:**

- One conversion path for every hypothesis test, so a cell the parametric tests refuse is refused everywhere, with the same message shape.
- The existing order of checks is kept. A length or emptiness error that fires today before any cell is read still fires first.
- The goroutine reads in `OneWayANOVA`, `KruskalWallis` and `FriedmanTest` stay parallel.

**Non-Goals:**

- Positions in errors outside the hypothesis tests. Regression numbers its predictors from zero (`LinearRegression(y, x1, x2)` with a blank in `x2` reports `predictor 1`), its domain checks print a zero-based `(index N)`, and factor analysis, PCA and correlation number columns from zero. These go into an `AGENTS.md` follow-up.
- `NaN` acceptance outside the hypothesis tests. `PCA` returns `NaN` eigenvalues with a nil error, and `KNNRegress` predicts `NaN` from a `NaN` target. Both are recorded in the same follow-up.
- A typed-nil list passed to `numericSlice` directly, which regression still reaches, and the other `stats` functions that call `AtomicDo` on an argument without converting it. They stay in the existing nil-list follow-up.
- Reading the groups of `OneWayANOVA`, `KruskalWallis` and `FriedmanTest` atomically as one snapshot. Each group is read under its own lock, as today.

## Decisions

### A position word, not only `row`

`numericValues` always writes `row`. That is right for a list whose cells are observations, and wrong for `FriedmanTest` and `RepeatedMeasuresANOVA`, where each list is one subject and its cells are that subject's conditions. In a table laid out one subject per row, "row 3" would point at the wrong axis.

The change adds `appendNumericValues(dst, raw, label, position)`, which writes `position` in place of `row`, and makes `numericValues` call it with `"row"`. Every existing caller keeps its message byte for byte.

The alternative was to call conditions rows and explain it in the documentation. It was rejected because the error is where the reader needs the right word.

### The label is built only when a cell is refused

The multi-group functions first called `numericValues` once per group with a label from `fmt.Sprintf` and copied the returned slice into their flat values slice. Measured on the M3, interleaving ten 2-second runs of each build, that made `BenchmarkTwoWayANOVA` about 20% slower (median 33.4 µs before, 40.2 µs after). A micro-benchmark of the conversion alone put the cause in the eager label and the extra slice: 46 allocations per call against 12, and 17.9 µs against 13.2 µs.

`appendNumericValues` therefore takes the label as a function that it calls only when it refuses a cell, and appends to the caller's slice instead of returning a new one. In the same micro-benchmark this measured 9 allocations and 13.4 µs. `numericValues` wraps its fixed label in such a function, so its callers do not change.

### Labels

| Function | List label | Position |
| --- | --- | --- |
| `SingleSampleWilcoxon` | `data` | `row` |
| `PairedTTest`, `PairedWilcoxon`, `MannWhitneyU` | `data1`, `data2` | `row` |
| `OneWayANOVA`, `KruskalWallis`, `LeveneTest`, `BartlettTest` | `group N` | `row` |
| `TwoWayANOVA` | `cell (A=a, B=b)` | `row` |
| `RepeatedMeasuresANOVA`, `FriedmanTest` | `subject N` | `condition` |

`N`, `a` and `b` count from one. `data`, `data1` and `data2` match the labels `SingleSampleTTest` and `TwoSampleTTest` already use, so every one- and two-sample test names its inputs the same way.

The other position-bearing errors in these functions are renumbered with no change to their wording: `group N is empty`, `empty cell at A=a, B=b`, `inconsistent condition count at subject N`, `subject N has M observations, expected K`, `group N is empty after rank assignment` and `group N must have at least two observations and positive variance`.

### Nil lists become empty lists before any goroutine starts

Each function converts its arguments with `asDataList` first, before it starts a goroutine or takes a lock. That is the existing convention in `stats`: the documented purpose of `asDataList` is that the caller's own checks report a nil list. All nine functions already refuse an empty sample, group, cell or subject, so a nil list is refused by a message that exists today. None of them treats a nil list as a legitimate empty sample, so the open question in the follow-up, whether a nil list is an error or an empty sample, has the same answer either way for these functions.

The alternative, an explicit `group N is nil` error, would add a second message for a case the empty check already covers.

### A wrapped list is read through its own `*DataList`, not rebuilt

`asDataList` used to copy any implementation other than `*insyra.DataList` with `insyra.NewDataList(dl.Data()...)`. `NewDataList` flattens every slice, so a wrapped list, such as one from `isr.DL`, turned a slice cell into several numeric cells and the tests counted them as observations. The review of this change found it, because routing six more functions through `asDataList` would have spread it to them. It also copied the list on the caller's goroutine, which made the parallel reads in `OneWayANOVA` and `KruskalWallis` serial for such lists: a reviewer measured `OneWayANOVA` on five 20,000-value `isr` lists going from 2.3 ms to 5.2–6.9 ms.

`asDataList` now takes the `*insyra.DataList` the implementation's `AtomicDo` hands over and uses it directly. Every implementation in the repository embeds a `*insyra.DataList`, so that is the list itself, and its cells arrive as stored. No `stats` caller mutates the list `asDataList` returns, so returning the live list instead of a copy is safe, and it is what the concrete branch already did. Measured after the change on the same `isr` input, interleaving five runs of each build: `OneWayANOVA` 2.13 ms on the base commit and 2.18 ms after, `KruskalWallis` 12.6 ms and 10.8 ms, with the same bytes allocated.

The alternative was a non-flattening copy (`NewDataList().AppendDataList(dl)`). It fixes the flattening but keeps the serial copy.

### Conversion stays out of the lock

The paired and two-sample functions keep their `insyra.AtomicDoAll` snapshot, including the length check inside it, and run `numericValues` on the snapshot afterwards. This is what `testSeriesPair` does, and it is why that helper exists: validation allocates and formats errors, and should not hold two actors while it does. `PairedTTest` and `PairedWilcoxon` do not switch to `testSeriesPair`, because their length and emptiness checks have to run before any cell is converted, and `testSeriesPair` converts first.

### Result arithmetic is untouched

Only the conversion loops change. Each loop filled a `[]float64` or a flat values/labels pair in the same order the new code does, so the numbers reaching the arithmetic are the same values in the same order, and the results on finite input are bit-identical. The existing R-verified tests pin this and must pass unmodified. `TwoWayANOVA` accumulates its per-cell sums during conversion. It now sums over the values just appended for that cell, in the same order.

## Risks / Trade-offs

- [Code that matched the old error text breaks] → The text had no position or a zero-based one and differed per function, so matching it was fragile already. The CHANGELOG entry quotes the new form so a caller can update their match. Nothing in this repository matches the old text.
- [The new check slowing the multi-group tests] → Measured, and it did: see "The label is built only when a cell is refused". `BenchmarkTwoWayANOVA` is compared against the base commit again after that fix, and the numbers are recorded in the tasks.
- [Error precedence shifts when both inputs of a paired test are bad] → The old loop checked `data1[i]` and `data2[i]` row by row. The new code checks all of `data1` first. A pair with a bad cell at row 4 of `data1` and row 2 of `data2` now reports `data1` at row 4. Either answer is correct, and the one-sample and two-sample t-tests already behave this way.
