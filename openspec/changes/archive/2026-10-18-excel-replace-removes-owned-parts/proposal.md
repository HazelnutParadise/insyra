# Proposal: excel-replace-removes-owned-parts

## Why

Replacing a sheet deleted the worksheet and its relationships but left the parts they pointed to (comments, legacy drawing, tables, drawings) in the saved file, referenced by nothing. Measured on this line: the old comment text was still inside the file, and a new table could not take the old table's name (`the same name table already exists`).

## What Changes

- `excelsheet.Replace` removes the parts the old sheet owned, transitively, unless another part still references them, together with their content-type entries.
- Verification is by reading the saved package and reopening it with excelize. Excel and LibreOffice were not available.

## Capabilities

### Modified Capabilities

- `excel-write`: a replaced sheet leaves nothing of the old sheet in the file.

## Impact

`internal/excelsheet/`, `Docs/DataTable.md`, `Docs/csvxl.md`, both CHANGELOGs, `AGENTS.md` follow-up removed.
