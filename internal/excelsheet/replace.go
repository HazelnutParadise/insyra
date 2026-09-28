// Package excelsheet holds the workbook operations the root package and csvxl
// share, so a sheet is replaced one way everywhere.
package excelsheet

import "github.com/xuri/excelize/v2"

// Replace empties the sheet at idx by deleting and re-creating it, then moves
// it back to where it was and restores the active sheet. Nothing of the old
// sheet survives: no cell, formula, hidden row, row height, comment, hyperlink
// or column width. excelize refuses to delete a workbook's only sheet, so that
// case goes through a placeholder. Sheet names match without regard to case,
// as in Excel.
func Replace(f *excelize.File, sheet string, idx int) error {
	sheets := f.GetSheetList()
	active := f.GetSheetName(f.GetActiveSheetIndex())
	next := ""
	if idx+1 < len(sheets) {
		next = sheets[idx+1]
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
	}
	activeIdx, err := f.GetSheetIndex(active)
	if err != nil {
		return err
	}
	f.SetActiveSheet(activeIdx)
	return nil
}
