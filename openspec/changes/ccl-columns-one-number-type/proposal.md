# Proposal: ccl-columns-one-number-type

## Why

`ccl-exact-integer-arithmetic` made a literal written as digits alone an `int64`, so a literal fallback put integers into a float column: `COALESCE(TONUM(A), 0)` over `["19.5", "abc", "30"]` gave `[19.5, 0 (int64), 30]`, measured on 2026-10-04. The values were right and `Show`, `ToJSON` and `ToStringSlice` printed them the same, but a caller reading the cell as a `float64` failed on that row, and the docs had to tell users to write `0.0`. The owner set the goal on 2026-10-04: users should not have to notice types. pandas reaches it with one dtype per column.

## What Changes

- A column `AddColUsingCCL`, `EditColByIndexUsingCCL`, `EditColByNameUsingCCL` or a statement of `ExecuteCCL` writes holds one kind of number. When its numbers include a `float64` or `float32`, every integer a `float64` holds exactly becomes a `float64`.
- An integer past 2^53 keeps its digits, as the owner required ("unless the number is outside what the type can represent").
- A column whose numbers are all integers is left as it is, narrow Go types included. Values that are not numbers (text, booleans, `nil`, dates, durations, decimals) are never touched; in a column mixing text and numbers only the numbers are settled.
- `Docs/CCL.md` drops the advice to write `0.0`; its `COALESCE(TONUM(...), 0)` example is a `float64` column again.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ccl-type-semantics`: a computed column holds one kind of number.

## Impact

- `ccl_number_types.go` (new, `unifyCCLNumbers`), called from `applyCCLOnDataTable` in `ccl.go` and from `executeAssignment` and `executeNewColumn` in `datatable_ccl.go`. The CLI, the DSL and `isr` reach CCL through these methods. `parquet.ApplyCCL` already writes a column mixing integers and floats as `float64`, so in memory and on disk now agree.
- Results change only for a computed column that mixed integers and floats: its integers up to 2^53 become `float64`s of the same value.
- `Docs/CCL.md`, `skills/insyra/SKILL.md`, the `ccl-exact-integer-arithmetic` entry of both changelogs, `delivery-status.md`.
