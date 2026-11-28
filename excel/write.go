package excel

import (
	"fmt"
	"io"

	"github.com/HazelnutParadise/insyra"
)

// ErrSheetExists reports that Write was asked to write a sheet the workbook
// already holds without SheetExistsReplace. Match it with errors.Is. It is
// the same error value as insyra.ErrSheetExists.
var ErrSheetExists = insyra.ErrSheetExists //nolint:staticcheck // the same value as the deprecated core name until 0.4

// SheetExistsPolicy says what Write does when the workbook already holds the
// sheet it is writing. It is the same type as insyra.SheetExistsPolicy.
type SheetExistsPolicy = insyra.SheetExistsPolicy //nolint:staticcheck // the same type as the deprecated core name until 0.4

const (
	// SheetExistsFail refuses the save and leaves the file exactly as it was.
	// It is the zero value, so a sheet is never overwritten unless asked.
	SheetExistsFail = insyra.SheetExistsFail //nolint:staticcheck // the same constants as the deprecated core names until 0.4
	// SheetExistsReplace discards the sheet, formatting included, and writes
	// the table in its place, keeping the sheet's position among the others.
	SheetExistsReplace = insyra.SheetExistsReplace //nolint:staticcheck // the same constants as the deprecated core names until 0.4
)

// WriteOptions configures Write and WriteTo. The zero value writes the sheet
// "Sheet1" with a row of column names and without row names, and refuses to
// overwrite a sheet that is already there.
type WriteOptions struct {
	// Sheet is the sheet the table is written to. Empty means "Sheet1".
	Sheet string
	// NoHeaderRow leaves out the row of column names, which is written by
	// default.
	NoHeaderRow bool
	// HasRowNames writes the row names as the first column.
	HasRowNames bool
	// IfSheetExists is what Write does when the workbook already holds Sheet.
	// WriteTo always writes a new workbook and ignores it.
	IfSheetExists SheetExistsPolicy
}

func (opt WriteOptions) core() insyra.ExcelWriteOptions { //nolint:staticcheck // the deprecated core options are this package's implementation until 0.4
	return insyra.ExcelWriteOptions{ //nolint:staticcheck // the deprecated core options are this package's implementation until 0.4
		Sheet:                 opt.Sheet,
		SetColNamesToFirstRow: !opt.NoHeaderRow,
		SetRowNamesToFirstCol: opt.HasRowNames,
		IfSheetExists:         opt.IfSheetExists,
	}
}

// Write writes dt as one sheet of the workbook at path. A missing workbook is
// created; an existing one keeps every other sheet as it was. When the
// workbook already holds the sheet, the save fails with ErrSheetExists and
// the file is not touched, unless opts.IfSheetExists is SheetExistsReplace.
// opts are the settings WriteOptions describes; at most one may be given.
//
// The workbook is written to a temporary file that is renamed into place, so
// a failure never leaves a damaged workbook behind. Numbers, booleans and
// times are written as Excel values; a time keeps its wall-clock reading but
// not its time zone, which Excel does not store.
func Write(dt insyra.IDataTable, path string, opts ...WriteOptions) error {
	opt, err := oneOption("WriteOptions", opts)
	if err != nil {
		return err
	}
	if insyra.IsNilTable(dt) {
		return fmt.Errorf("the table is nil")
	}
	return dt.ToExcel(path, opt.core()) //nolint:staticcheck // the deprecated writer is this function's implementation until 0.4
}

// WriteTo writes dt as a one-sheet workbook to any destination — an HTTP
// response, a zip entry, a buffer — the same way Write writes a new file. It
// does not close w.
func WriteTo(dt insyra.IDataTable, w io.Writer, opts ...WriteOptions) error {
	opt, err := oneOption("WriteOptions", opts)
	if err != nil {
		return err
	}
	if insyra.IsNilTable(dt) {
		return fmt.Errorf("the table is nil")
	}
	return dt.WriteExcel(w, opt.core()) //nolint:staticcheck // the deprecated writer is this function's implementation until 0.4
}
