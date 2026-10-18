// Package excelsheet holds the workbook operations the root package and csvxl
// share, so a sheet is replaced one way everywhere.
package excelsheet

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Replace empties the existing sheet at idx by deleting and re-creating it,
// then moves it back to where it was, hides it again if it was hidden and
// restores the active sheet. What lays the sheet out without touching its
// data is carried over: the widths of its visible columns and its view
// (zoom, gridlines, frozen or split panes). Nothing else stored in the old
// sheet survives: no cell, formula, hidden row or column, row height, comment,
// hyperlink or merged range, and the comments, tables, drawings and pictures
// it owned are removed from the file too, unless another part still uses
// them. Hidden rows and columns and merged ranges are dropped because they
// would hide the new data.
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
	layout := readLayout(f, sheet)
	// Read what the old sheet owns before DeleteSheet drops its relationships.
	owned := ownedParts(f, idx)
	if err := f.DeleteSheet(sheet); err != nil {
		return err
	}
	dropParts(f, owned)
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}
	if err := layout.apply(f, sheet); err != nil {
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

// excelizeDefaultColWidth is the width GetColWidth reports for a column that
// sets none, so a column at that width has nothing to carry over.
const excelizeDefaultColWidth = 9.140625

// layout is what a sheet records about how it looks apart from its cells.
type layout struct {
	widths map[string]float64
	view   *excelize.ViewOptions
	panes  *excelize.Panes
}

// readLayout reads the widths of the sheet's visible columns and its first
// view and panes. Every column Excel has is asked, because the sheet's stored
// dimension only covers the cells written when it was loaded and says nothing
// of a column that has a width and no cell. A part that cannot be read is left
// out, since losing a width is not worth failing the replace.
func readLayout(f *excelize.File, sheet string) layout {
	l := layout{widths: map[string]float64{}}
	for c := 1; c <= excelize.MaxColumns; c++ {
		name, err := excelize.ColumnNumberToName(c)
		if err != nil {
			break
		}
		if w, err := f.GetColWidth(sheet, name); err != nil {
			break
		} else if w != excelizeDefaultColWidth {
			if visible, err := f.GetColVisible(sheet, name); err == nil && visible {
				l.widths[name] = w
			}
		}
	}
	if view, err := f.GetSheetView(sheet, 0); err == nil {
		l.view = &view
	}
	if panes, err := f.GetPanes(sheet); err == nil && (panes.Freeze || panes.Split) {
		l.panes = &panes
	}
	return l
}

// apply writes the layout onto the freshly created sheet.
func (l layout) apply(f *excelize.File, sheet string) error {
	for name, w := range l.widths {
		if err := f.SetColWidth(sheet, name, name, w); err != nil {
			return err
		}
	}
	if l.view != nil {
		if err := f.SetSheetView(sheet, 0, l.view); err != nil {
			return err
		}
	}
	if l.panes != nil {
		if err := f.SetPanes(sheet, l.panes); err != nil {
			return err
		}
	}
	return nil
}
