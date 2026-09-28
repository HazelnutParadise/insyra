# Design

## Context

`Replace` deletes the sheet, creates it again at the end, and moves it back before the sheet that followed it. Position and the active tab are restored already. Everything else the workbook records about sheets lives in `workbook.xml` too: the `state` attribute that hides a sheet, and the `localSheetId` that ties a defined name, a print area or an autofilter range to its sheet by index.

## Decisions

- **Raise the shifted names back, directly.** excelize's `DeleteSheet` lowers the `localSheetId` of names above the deleted index; `NewSheet` appends; `MoveSheet` moves the sheet back without touching any `localSheetId`. After the move the rebuilt sheet sits at its old index with no names of its own (`DeleteSheet` removed them), so every sheet-scoped name at that index or above is one too low. `Replace` adds one to each, through `f.WorkBook`, excelize's exported workbook. Re-creating the names through `DeleteDefinedName`/`SetDefinedName` was rejected: `DefinedName` carries only name, comment, reference and scope, so `hidden` and the other attributes of a name such as `_xlnm._FilterDatabase` would be lost.
- **Keep the hidden state like the position.** The old `state` is read before the delete and written back after the move. `SetSheetVisible` was rejected because it checks the active sheet and cannot say "put back what was there".
- **Names defined for the replaced sheet alone still go.** They described the old sheet's content (its print area, its filter range), and `DeleteSheet` removes them. This is documented rather than changed.

This is the same code as `dev` 851f3114.
