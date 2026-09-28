// Package excelsheet holds the workbook operations the root package and csvxl
// share, so a sheet is replaced one way everywhere.
package excelsheet

import "github.com/xuri/excelize/v2"

// Replace empties the existing sheet at idx by deleting and re-creating it,
// then moves it back to where it was, hides it again if it was hidden and
// restores the active sheet. Nothing stored in the old sheet survives: no
// cell, formula, hidden row, row height, comment, hyperlink, column width,
// view or merged range. Names defined for a later sheet stay with that sheet;
// only the names belonging to the replaced sheet are deleted with it.
// excelize refuses to delete a workbook's only sheet, so that case goes
// through a placeholder. Sheet names match without regard to case, as in
// Excel.
func Replace(f *excelize.File, sheet string, idx int) error {
	sheets := f.GetSheetList()
	active := f.GetSheetName(f.GetActiveSheetIndex())
	next := ""
	if idx+1 < len(sheets) {
		next = sheets[idx+1]
	}
	// Whether the sheet is hidden is recorded in the workbook, not in the
	// sheet, so it is kept like the sheet's position.
	state := ""
	if wb := f.WorkBook; wb != nil && idx < len(wb.Sheets.Sheet) {
		state = wb.Sheets.Sheet[idx].State
	}
	placeholder := ""
	if f.SheetCount == 1 {
		placeholder = "__insyra_placeholder__"
		if _, err := f.NewSheet(placeholder); err != nil {
			return err
		}
	}
	if err := f.DeleteSheet(sheet); err != nil {
		return err
	}
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}
	if placeholder != "" {
		if err := f.DeleteSheet(placeholder); err != nil {
			return err
		}
	}
	if next != "" {
		if err := f.MoveSheet(sheet, next); err != nil {
			return err
		}
		// DeleteSheet moves every name defined for a later sheet down one
		// index, and MoveSheet does not move it back, so each of those names
		// would now belong to the sheet before its own. The rebuilt sheet is
		// back at idx with no names of its own, so every sheet-scoped name at
		// idx or above belongs one sheet further on.
		if wb := f.WorkBook; wb != nil && wb.DefinedNames != nil {
			for i := range wb.DefinedNames.DefinedName {
				if id := wb.DefinedNames.DefinedName[i].LocalSheetID; id != nil && *id >= idx {
					shifted := *id + 1
					wb.DefinedNames.DefinedName[i].LocalSheetID = &shifted
				}
			}
		}
	}
	if wb := f.WorkBook; state != "" && wb != nil && idx < len(wb.Sheets.Sheet) {
		wb.Sheets.Sheet[idx].State = state
	}
	activeIdx, err := f.GetSheetIndex(active)
	if err != nil {
		return err
	}
	f.SetActiveSheet(activeIdx)
	return nil
}
