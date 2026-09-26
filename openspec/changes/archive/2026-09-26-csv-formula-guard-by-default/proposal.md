## Why

#285 (SEC-3) added a CSV formula guard as `CSVWriteOptions.SanitizeFormulas`, off by default, and left the default to the owner. The owner ruled on 2026-09-26 to turn it on. Measuring it first showed a defect: with the guard on, a number such as `-5` was written `'-5`, which a spreadsheet then reads as text and a program reads back as a string. A number is never a formula, and neither is text that is only a number (`+886912345678`), so the guard now leaves both alone, which also removes most of what made a default-on guard costly.

## What Changes

- **BREAKING (output)**: `ToCSV` and `WriteCSV` guard by default: text starting with `=`, `+`, `-` or `@` that is not only a number gets a leading single quote.
- `CSVWriteOptions.AllowFormulas` turns the guard off, for a file read back by a program. Named so the zero value is the guarded, common case.
- `SanitizeFormulas` (shipped in v0.3.3) stays one release as a Deprecated no-op.
- CLI `save` takes `allowformulas true|false` for CSV, default `false`.

## Capabilities

### New Capabilities
- `csv-formula-guard`: what CSV output does with text a spreadsheet would run.

### Modified Capabilities
None.

## Impact

- `datatable_csv.go`, `cli/commands/save.go`; tests `csv_formula_guard_test.go`, `batch6_test.go`, `cli/commands/load_save_test.go`.
- `Docs/DataTable.md`, `Docs/cli-dsl.md`, the CLI skill, both changelogs, `AGENTS.md` follow-up.
