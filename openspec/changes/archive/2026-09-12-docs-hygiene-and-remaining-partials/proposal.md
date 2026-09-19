# Proposal: docs-hygiene-and-remaining-partials

## Why

What is left of the decision-free backlog: #232 (T-22), #274 (Q-10), #281 (RP-9), #282 (RP-14), the version-pinning half of #280 (RP-6), and the parts of #269, #273, #329 and #244 that do not need a ruling.

- **Nineteen exported `DataTable` methods had no doc comment**, and two doc comments named the wrong function: `AppendRowsByColIndex`'s said `AppendRowsByIndex`, and `ToJSON_Bytes` recorded its errors under `ToJSON_Byte`.
- **`parquet`'s doc comments used a `Read: read …` colon style** rather than Go's `Read reads …`, and `FileInfo`, `ColumnInfo` and `RowGroupInfo` had none.
- **`.gitignore` listed three directory names that no longer exist.**
- **The skill had three reference files**, none for `stats` or `plot`, the two packages it is asked about most.
- **CI mixed `checkout@v4`/`@v5` and `setup-go@v5`/`@v6`, and pinned golangci-lint to `latest`**, so a lint result was not reproducible and half the jobs still ran on the deprecated Node 20.
- **`ExcelToCsv` dropped a named sheet that was not in the file**, so a typo produced a smaller conversion that looked like it had worked, and `excelFileToCsv` logged under its caller's name.
- **`read <file> as x` answered `unknown option "as"`.** `read` appends its own alias before handing the arguments to `load`, so a user-supplied one arrived as a second `as` and the message pointed at the wrong thing.
- **`Stream`'s drain-or-cancel contract was not written down**, and neither was `SingleSampleTTest`'s behaviour on constant data or the z-tests' unsigned effect size.

## What Changes

- Doc comments for the nineteen methods and the three parquet types; the two wrong names corrected, including the one that reached `Err()`.
- `parquet`'s comments are rewritten in Go's form.
- The three stale `.gitignore` lines go.
- `references/stats.md` and `references/plotting.md` are added and listed in `SKILL.md`, along with the `ml-decision-tree.md` entry that was already there but unlisted. They cover picking a test, what each result type holds, which chart takes what data shape, which save path needs a browser, and the places each package answers something surprising.
- Every workflow uses `checkout@v5` and `setup-go@v6`, and golangci-lint is pinned to `v2.12.2`.
- `ExcelToCsv` reports a named sheet the file does not have, listing the ones it does. `excelFileToCsv` logs under its own name.
- `read` refuses `as` with a message that says what to use instead.
- `Stream`, `SingleSampleTTest` and both z-tests document the behaviour that surprised the review.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `io-error-hygiene`: a named sheet that is not there is an error.
- `verification-integrity`: CI pins the versions it runs.

## Impact

- `datatable.go`, `datatable_colname.go`, `datatable_json.go`, `parquet/api.go`, `stats/ttest.go`, `stats/ztest.go`, `csvxl/convert.go`, `csvxl/convertDir.go`, `cli/commands/read.go`, `.gitignore`, the seven workflows, `skills/insyra/`.
- User-visible in two places, so both changelogs get an entry: a missing sheet is now an error, and `read … as x` explains itself.
- **Deliberately not done, with reasons.** RP-7 of #280 (extract the duplicated Python/R install steps into a composite action) is decision-free but touches all three verification workflows at once, and RP-8 (which extra linters to enable) is a ruling. The other halves of #269 (whether `.csv` should be appended automatically), #273 (moving `Stream` to `iter.Seq2`), #329 (whether `save` should write `.xlsx`), #244 (the effect-size sign, already decided as deliberate in batch 2) and #233 (telling `DataTableSortConfig{}` from `{ColumnNumber: 0}`, which needs the struct to change) all need an owner decision.

## Backport to dev (0.3.x)

Dev received: the doc comments for the nineteen `DataTable` methods, reworded where they described 0.4's error model (on this line a failure logs a warning and replaces `Err()`), and `Data`'s comment says its slices share storage with the table, because on this line they are not copies; `ToJSON_Bytes` recording its error under its own name through `dt.warn`; the `parquet` comments, with `Write`'s describing this line's direct write rather than 0.4's temporary file and rename; the `.gitignore` cleanup; `checkout@v5`, `setup-go@v6` and golangci-lint v2.12.2 in every workflow; `excelFileToCsv`'s log name; `read … as <var>` guidance; the `SingleSampleTTest` and z-test comments (the constant-data behaviour was checked on this line); `references/stats.md`, `references/plotting.md` and the `SKILL.md` list, checked against this line's code, with eight claims from 0.4's text corrected (`SavePNG`'s online fallback is the default, `gplot.SaveChart` returns nothing, `CreateHeatmapChart`'s row rule and accepted types, `gplot`'s extensions; which `stats` functions return an error, `confidenceLevel` outside `(0, 1)` being an error, `CutTreeByK`'s error return, and which regressions take options); the two CLI skill notes; and the `verification-integrity` Purpose and requirement.

Left on 0.4:
- `ExcelToCsv` refusing a name in `onlyContainSheets` the workbook lacks, its `Docs/csvxl.md` paragraph, changelog entry and the `io-error-hygiene` requirement: breaking, a call that succeeds today would return an error;
- the AGENTS.md follow-up counting 0.4's archived specs with no written Purpose, which describes 0.4's spec set;
- `api-review.md` and `delivery-status.md` edits.

