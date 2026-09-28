# Design

## Context

On `0.4`, 7e9b431e moved `DataTable.ToExcel`'s sheet replacement into `internal/excelsheet.Replace` so that `csvxl.AppendCsvToExcel` and `ToExcel` replace a sheet one way. This line has no `DataTable.ToExcel` (it came with `0.4`'s Excel writer), so `csvxl` is the only caller.

## Decisions

- **The routine lives in `csvxl/convert.go`, not in a new internal package.** Its body is `0.4`'s `excelsheet.Replace` unchanged: record the sheet list, the active sheet and the sheet after the target; add a placeholder sheet when the target is the workbook's only sheet, because excelize refuses to delete the last sheet; delete and re-create the target; delete the placeholder; move the target back before the sheet that followed it; restore the active sheet. With one caller, a package exists only to hold one function. If `ToExcel` comes to this line, it moves into `internal/excelsheet` as on `0.4`.
- **What the workbook records about the sheet stays; what the sheet stores goes.** Position, hidden state and the active tab live in `workbook.xml`, as do the defined names; column widths, views, merged ranges, row state, comments and links live in the worksheet. `0.4`'s routine keeps position and the active tab. Two more workbook-level things break with it as written, measured on this port: `DeleteSheet` lowers the `localSheetId` of every name defined for a later sheet and `MoveSheet` does not raise it again, so on First, Target, Last a name `Rate` of `Last` moved to `Target` and `Last!C1 = Rate*10` became `#NAME?`; and a hidden sheet came back visible. After moving the rebuilt sheet back, every sheet-scoped name at its index or above is raised by one (the rebuilt sheet has no names of its own, because `DeleteSheet` removed them), and the old `state` attribute is written back. Both go through `f.WorkBook`, excelize's exported workbook, because re-creating the names through `SetDefinedName` would lose attributes such as `hidden` that `DefinedName` does not carry. `0.4`'s `excelsheet.Replace` has the same two defects.
- **`paddedSheetCells` goes.** It existed only to list the cells of a sheet whose cells carry no address, for clearing; a rebuilt sheet needs no list.

## Risks / Trade-offs

- A caller who formatted a sheet and relies on `AppendCsvToExcel` to keep its column widths, views or merged ranges loses them. That was the v0.3.3 promise. It is withdrawn on the owner's direction to use `0.4`'s fix as it is: data that reads back correctly comes before the old sheet's formatting. Copying the column widths onto the rebuilt sheet was possible; for everything the worksheet itself stores, this line follows `0.4`.
