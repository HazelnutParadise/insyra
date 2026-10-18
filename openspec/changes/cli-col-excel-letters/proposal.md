# Proposal: cli-col-excel-letters

## Why

[#438](https://github.com/HazelnutParadise/insyra/issues/438): `insyra col t A` fails with `column not found: A`, although Excel-style letters address columns everywhere else (`GetCol("A")`, CCL, the `stats` table functions). `col` took only a name or a 0-based number.

## What Changes

- `col <var> <name|number|letter>` resolves a selector in this order: a whole number is a 0-based position (unchanged); an exact column name; an Excel-style letter that lands inside the table.
- Name comes before letter, so every call that worked keeps its meaning, including a column literally named `A` that is not column A. The `stats` table functions resolve the letter first; matching them would change working `col` calls, so this line keeps names first and the rule is written in the usage text.
- A letter past the last column is "column not found".

## Capabilities

### Modified Capabilities

- `dsl-commands`: `col` accepts Excel-style letters.

## Impact

`cli/commands/col.go`, a test in `cli/commands/batch4_test.go`, `Docs/cli-dsl.md`, both CHANGELOGs.
