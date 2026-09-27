# Proposal: hypothesis-tests-refuse-non-finite

## Why

Nine hypothesis tests in `stats` accept a `NaN` or `±Inf` cell and carry it into the arithmetic, and the answer that comes back looks like any other. Measured on `dev` at 550cf94e on 2026-09-27, with one `NaN` among otherwise clean samples:

| Function | What it returns |
| --- | --- |
| `PairedTTest`, `OneWayANOVA`, `TwoWayANOVA`, `RepeatedMeasuresANOVA` | a `NaN` statistic and a `NaN` p-value, with a nil error |
| `KruskalWallis` on `[1, 2, NaN, 4]` against `[1, 2, 3, 4]` | H = 0.540, p = 0.462, with a nil error |
| `MannWhitneyU`, `SingleSampleWilcoxon`, `PairedWilcoxon`, `FriedmanTest` | a finite statistic and p-value, with a nil error |

The rank tests are the worst case: a `NaN` still receives a rank, so the result is finite and nothing downstream can tell it apart from a real one. `+Inf` and `-Inf` behave the same way.

The released `stats` documentation says that a value which cannot be read as a finite number is never replaced with a substitute. The t-, z- and F-tests, Levene, Bartlett, skewness, kurtosis, correlation and regression all refuse such a cell with `<list> contains a non-finite value at row N`. These nine functions check only whether a cell converts, so they are the remaining exception.

Their errors also disagree about positions. Blank or text cells give `invalid numeric value in data1` with no position at all, or `invalid data at group 0 index 2`, `invalid data at cell (A=1, B=0) index 1` and `invalid numeric value at subject 1 condition 1`, all counted from zero. `LeveneTest` and `BartlettTest` mix the two conventions in one message: `group 1 contains a non-finite value at row 2` names the second group and the second row. A reader cannot paste either number into a spreadsheet without knowing which function produced it.

Three of them end the whole program when a group is a nil list. `OneWayANOVA`, `KruskalWallis` and `FriedmanTest` dereference each list inside a goroutine they start, so a typed-nil `*DataList` crashes the process with a nil-pointer panic that no caller can recover. `TwoWayANOVA`, `RepeatedMeasuresANOVA` and `SingleSampleWilcoxon` panic in the caller's goroutine. This is the `stats` nil-list entry in `AGENTS.md`'s follow-ups.

## What Changes

- `PairedTTest`, `SingleSampleWilcoxon`, `PairedWilcoxon`, `MannWhitneyU`, `OneWayANOVA`, `TwoWayANOVA`, `RepeatedMeasuresANOVA`, `KruskalWallis` and `FriedmanTest` check every cell the way the other tests do. A blank, text, `NaN`, `+Inf` or `-Inf` cell is refused with `<where> contains a non-numeric value at <position>: <value>` or `<where> contains a non-finite value at <position>: <value>`.
- Every position in an error from a hypothesis test counts from one and says what it counts. A list is named `data`, `data1`, `data2`, `group 2`, `cell (A=2, B=1)` or `subject 3`, and the cell inside it is a `row` or, for the subject-by-condition tests, a `condition`. This also renumbers `LeveneTest` and `BartlettTest`'s group, and the other errors these functions raise about a group, a cell or a subject: an empty group, an empty cell, a subject with the wrong number of conditions, a Bartlett group with too few observations.
- A nil list, typed or not, reads as an empty list in all nine functions. Each of them already refuses an empty group, cell, subject or sample, so a nil list now returns that error instead of panicking or ending the program.
- A list that is not a `*insyra.DataList` itself, such as one from `isr.DL`, is read as it is stored. `asDataList` used to rebuild it with `NewDataList`, which flattens a slice cell into several numbers. Found in review: routing six more functions through `asDataList` would have let such a list pass a slice cell as extra observations, which base refused, and `SingleSampleTTest` and `PairedTTest` already did (a four-cell list paired with a four-cell one reached `PairedTTest` as five cells and failed the length check).
- Results on fully finite input are unchanged, bit for bit.
- `Docs/stats.md` states one treatment for every hypothesis test and replaces the table of per-function messages. `skills/insyra/SKILL.md` stops saying that `NaN` and `±Inf` pass through statistics functions in general, and names the two places they still do. Both CHANGELOGs get a `stats` entry.
- `AGENTS.md`: the nil-list follow-up shrinks to the functions this change does not touch, and a new follow-up records the zero-based positions and `NaN` acceptance measured outside the hypothesis tests.

### Release line

This goes to `dev`, the 0.3.x line, without a **BREAKING** mark. It is the same kind of change as the v0.3.3 entry that made `SingleSampleTTest`, `TwoSampleTTest`, the z-tests, the F-test, Levene, Bartlett and `CalculateMoment` refuse unreadable cells. That entry shipped on 0.3.x unmarked because the released documentation already said every numeric entry point refuses such a value. The owner's rule for the 0.3.x line, recorded on 2026-09-14, is that a fix bringing code into line with what v0.3 documentation already described belongs on 0.3.x even when it changes a result.

What a caller will notice:

- A call that used to return a `NaN` or a wrong finite number with a nil error now returns an error.
- The error text of these nine functions changes, and so does the group number in `LeveneTest` and `BartlettTest` errors. Code that matched the old text has to be updated. Nothing in this repository matches it; `cli`'s `ttest paired`, `anova` and `levene` commands print the error as it comes back and so show the new text.
- A nil list returns an error where it used to panic.

`0.4` has the same code, so it receives the change the ordinary way, when `dev` is next merged into `0.4`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-test-input`: two requirements are added. The paired, rank and ANOVA tests refuse unreadable cells and read a nil list as empty. Every hypothesis test numbers the positions in its errors from one and names what each number counts.

## Impact

- Code: `stats/numericinput.go`, `stats/asdatalist.go`, `stats/ttest.go`, `stats/nonparam_wilcoxon.go`, `stats/nonparam_mwu.go`, `stats/anova.go`, `stats/nonparam_kw.go`, `stats/nonparam_friedman.go`, `stats/ftest.go`.
- Tests: a new test file in `stats` covering `NaN`, `+Inf`, `-Inf`, blank and text cells and nil lists in each of the nine functions, and the one-based numbering in all eleven. Existing R-verified tests must pass unmodified.
- Docs: `Docs/stats.md`, `skills/insyra/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `AGENTS.md`.
- No API signature changes and no new dependencies.
