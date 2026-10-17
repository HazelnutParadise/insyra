# Proposal: core-go-names

## Why

K-13 and I-4 of the API review, ported from `0.4` to the 0.3.x line as renames with Deprecated wrappers, filed under [#212](https://github.com/HazelnutParadise/insyra/issues/212), found names in the core package and in `isr` that break Go's conventions. Part of the review is settled elsewhere: `isr`'s `FirstCol2RowNames` became `HasRowNames`, and the `csvxl` and `datafetch` names were cleaned up in their own changes. The owner ruled on the rest on 2026-10-12:

- `ParseColIndex` and `CalcColIndex` are a pair whose verbs do not mirror each other, so a reader cannot tell from one name what the other is called. The owner chose `ColIndexToNumber` and `ColNumberToIndex`. Their nouns are the ones the library already uses: an index is the Excel-style letters (`"AA"`), and a number is the 0-based position (`GetColIndexByNumber`, `GetColNumberByName`). `strconv`'s `ParseX`/`FormatX` pair was rejected because `Format` does not say what comes out. excelize's `ColumnNameToNumber` counts from 1, and these count from 0, so the doc comments state the base.
- `Config.GetDoesUseColoredOutput` stacks two verbs. It becomes `GetUseColoredOutput`, mirroring `SetUseColoredOutput`.
- The `Get` prefix stays across the library. Effective Go's "Getters" advice drops it for field getters, but insyra has 117 exported `Get…` names, all paired with a `Set…`. Dropping the prefix would break almost every caller for no change in behaviour, which is the reasoning of the earlier `Options`/`Config` ruling.
- `Config.Dangerously_TurnOffThreadSafety` keeps its underscore on purpose: a name that looks wrong is the warning. Naming linters report only the declaration, never a caller.
- `isr`'s `Of` and `From` are both kept on purpose, after Java's `List.of` and Rust's `From::from`.
- `isr`'s `Push` is kept, because neither `Push` is an alias. Measured on 7c6fc090: `dl.Push` is `Append` returning the `isr` wrapper so the chain continues, and `isr` cannot override `Append` and still satisfy `IDataList` (#208). `dt.Push(isr.Row{…})` adds a row, so `dt.Push` is not `AppendCols`.

The one-name rule of #211 says an old name stays one release as a Deprecated wrapper keeping its old meaning.

## What Changes

| Old name (Deprecated) | New name |
| --- | --- |
| `ParseColIndex` | `ColIndexToNumber` |
| `CalcColIndex` | `ColNumberToIndex` |
| `Config.GetDoesUseColoredOutput` | `Config.GetUseColoredOutput` |

- The new names return exactly what the old ones returned. Each old name stays one release as a Deprecated wrapper whose doc comment names its replacement.
- Every caller inside the module moves to the new names.
- `AGENTS.md` Key Conventions records three deliberate departures from Go convention: the `Get`/`Set` pairs, the underscore in `Dangerously_TurnOffThreadSafety`, and `isr`'s `Of`/`From` pair. It also records why `isr`'s `Push` is not a second name. The three old names are removed in 0.4.
- `Docs/isr.md` says how `Push` differs from `Append` and `AppendCols`, and why both `Of` and `From` exist.

## Capabilities

### New Capabilities

- `core-names`: the column index conversion names, the coloured-output getter, and the names kept against Go convention on purpose.

### Modified Capabilities

None.

## Impact

- `utils.go`, `config.go` and their in-module callers and tests.
- `Docs/utils.md`, `Docs/Configuration.md`, `Docs/DataTable.md`, `Docs/isr.md`, both changelogs and `AGENTS.md`.
- No behaviour changes. A caller of an old name gets a deprecation notice from staticcheck (SA1019), and nothing else.
