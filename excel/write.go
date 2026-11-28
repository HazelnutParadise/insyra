package excel

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/excelsheet"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/xuri/excelize/v2"
)

// ErrSheetExists reports that Write was asked to write a sheet the workbook
// already holds without SheetExistsReplace. Match it with errors.Is.
var ErrSheetExists = errors.New("sheet already exists")

// SheetExistsPolicy says what Write does when the workbook already holds the
// sheet it is writing.
type SheetExistsPolicy int

const (
	// SheetExistsFail refuses the save and leaves the file exactly as it was.
	// It is the zero value, so a sheet is never overwritten unless asked.
	SheetExistsFail SheetExistsPolicy = iota
	// SheetExistsReplace discards the sheet, formatting included, and writes
	// the table in its place, keeping the sheet's position among the others.
	SheetExistsReplace
)

// defaultSheet is the sheet a save writes when none is named. It is the name
// Excel gives the first sheet of a new workbook.
const defaultSheet = "Sheet1"

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

func (opts WriteOptions) sheetName() string {
	if opts.Sheet == "" {
		return defaultSheet
	}
	return opts.Sheet
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
	sheet := opt.sheetName()
	f, created, err := openOrNewWorkbook(path, sheet)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if !created {
		idx, err := f.GetSheetIndex(sheet)
		if err != nil {
			return err
		}
		switch {
		case idx == -1:
			if _, err := f.NewSheet(sheet); err != nil {
				return fmt.Errorf("failed to add sheet %s: %w", sheet, err)
			}
		case opt.IfSheetExists != SheetExistsReplace:
			return fmt.Errorf("%w: %s in %s", ErrSheetExists, f.GetSheetName(idx), path)
		default:
			if err := excelsheet.Replace(f, sheet, idx); err != nil {
				return fmt.Errorf("failed to replace sheet %s: %w", sheet, err)
			}
		}
	}

	if err := writeRows(dt, f, sheet, opt); err != nil {
		return err
	}
	return utils.WriteFileAtomically(path, func(w io.Writer) error {
		_, err := f.WriteTo(w)
		return err
	})
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
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := opt.sheetName()
	if err := f.SetSheetName(defaultSheet, sheet); err != nil {
		return fmt.Errorf("failed to name sheet %s: %w", sheet, err)
	}
	if err := writeRows(dt, f, sheet, opt); err != nil {
		return err
	}
	_, err = f.WriteTo(w)
	return err
}

// openOrNewWorkbook opens the workbook at path, or, when nothing is there,
// returns a new one whose only sheet is named sheet and reports it as
// created. Path is recorded on a new workbook so writing it checks the
// extension and sets the matching content type, as opening an existing one
// does.
func openOrNewWorkbook(path, sheet string) (*excelize.File, bool, error) {
	f, err := excelize.OpenFile(path, excelsheet.Options())
	if err == nil {
		return f, false, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, false, fmt.Errorf("failed to open Excel file %s: %w", path, err)
	}
	f = excelize.NewFile()
	f.Path = path
	if err := f.SetSheetName(defaultSheet, sheet); err != nil {
		_ = f.Close()
		return nil, false, fmt.Errorf("failed to name sheet %s: %w", sheet, err)
	}
	return f, true, nil
}

// writeRows writes the table into sheet, which must be empty. The names, row
// names and cells are read under one lock, so they come from one moment.
func writeRows(dt insyra.IDataTable, f *excelize.File, sheet string, opt WriteOptions) error {
	var rows [][]any
	dt.AtomicDo(func(t *insyra.DataTable) {
		names := t.ColNames()
		cells := t.To2DSlice()
		if !opt.NoHeaderRow {
			header := make([]any, 0, len(names)+1)
			if opt.HasRowNames {
				header = append(header, nil)
			}
			for _, name := range names {
				header = append(header, name)
			}
			rows = append(rows, header)
		}
		for rowIndex, cellRow := range cells {
			record := make([]any, 0, len(cellRow)+1)
			if opt.HasRowNames {
				rowName, _ := t.GetRowNameByIndex(rowIndex)
				record = append(record, rowName)
			}
			for _, v := range cellRow {
				record = append(record, cellValue(v))
			}
			rows = append(rows, record)
		}
	})
	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return err
		}
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			return fmt.Errorf("failed to write row index %d of sheet %s: %w", i, sheet, err)
		}
	}
	return nil
}

// cellValue passes the values Excel stores natively to excelize and writes
// everything else as the same text ToCSV and ToJSON would.
func cellValue(v any) any {
	switch v.(type) {
	case nil, string, bool, time.Time,
		int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return v
	}
	return utils.ValueText(v)
}
