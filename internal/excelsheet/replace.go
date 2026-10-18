// Package excelsheet holds the workbook operations the root package and csvxl
// share, so a sheet is replaced one way everywhere.
package excelsheet

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Replace empties the existing sheet at idx by deleting and re-creating it,
// then moves it back to where it was, hides it again if it was hidden and
// restores the active sheet. Nothing stored in the old sheet survives: no
// cell, formula, hidden row, row height, comment, hyperlink, column width,
// view or merged range, and the comments, tables, drawings and pictures it
// owned are removed from the file too, unless another part still uses them.
// Names defined for a later sheet stay with that sheet; only the names
// belonging to the replaced sheet are deleted with it.
// excelize refuses to delete a workbook's only sheet, so that case goes
// through a placeholder the workbook does not hold yet. Sheet names match
// without regard to case, as in Excel.
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
	// excelize's NewSheet returns the existing sheet when the name already
	// belongs to one, and DeleteSheet does nothing to a workbook's only sheet,
	// so a placeholder that collides with a sheet of the same name would leave
	// the old sheet in the file. Pick a name nothing holds yet and confirm a
	// new sheet really was added.
	placeholder := ""
	if f.SheetCount == 1 {
		placeholder = availableSheetName(f, "__insyra_placeholder__")
		before := f.SheetCount
		if _, err := f.NewSheet(placeholder); err != nil {
			return err
		}
		if f.SheetCount != before+1 {
			return fmt.Errorf("placeholder sheet %s was not created next to %s", placeholder, sheet)
		}
	}
	// Read what the old sheet owns before DeleteSheet drops its relationships.
	owned := ownedParts(f, idx)
	if err := f.DeleteSheet(sheet); err != nil {
		return err
	}
	dropParts(f, owned)
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

// availableSheetName returns base, or base with a running number appended, the
// first name no sheet of f holds yet. Excel matches sheet names without regard
// to case, so the collision check has to as well.
func availableSheetName(f *excelize.File, base string) string {
	name := base
	for i := 1; ; i++ {
		if idx, _ := f.GetSheetIndex(name); idx == -1 {
			return name
		}
		name = fmt.Sprintf("%s%d", base, i)
	}
}
