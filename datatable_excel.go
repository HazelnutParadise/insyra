package insyra

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/xuri/excelize/v2"
)

// ErrSheetExists reports that ToExcel was asked to write a sheet the workbook
// already holds without SheetExistsReplace. Match it with errors.Is.
var ErrSheetExists = errors.New("sheet already exists")

// SheetExistsPolicy says what ToExcel does when the workbook already holds the
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

// defaultExcelSheet is the sheet a save writes when none is named. It is the
// name Excel gives the first sheet of a new workbook.
const defaultExcelSheet = "Sheet1"

// ExcelWriteOptions configures ToExcel and WriteExcel.
type ExcelWriteOptions struct {
	// Sheet is the sheet the table is written to. Empty means "Sheet1".
	Sheet string
	// SetColNamesToFirstRow writes the column names as the first row.
	SetColNamesToFirstRow bool
	// SetRowNamesToFirstCol writes the row names as the first column.
	SetRowNamesToFirstCol bool
	// IfSheetExists is what ToExcel does when the workbook already holds
	// Sheet. WriteExcel always writes a new workbook and ignores it.
	IfSheetExists SheetExistsPolicy
}

func (opts ExcelWriteOptions) sheetName() string {
	if opts.Sheet == "" {
		return defaultExcelSheet
	}
	return opts.Sheet
}

// ToExcel writes the table as one sheet of the workbook at path. A missing
// workbook is created; an existing one keeps every other sheet as it was.
// When the workbook already holds the sheet, the save fails with
// ErrSheetExists and the file is not touched, unless opts.IfSheetExists is
// SheetExistsReplace.
//
// The workbook is written to a temporary file that is renamed into place, so
// a failure never leaves a damaged workbook behind. Numbers, booleans and
// times are written as Excel values; a time keeps its wall-clock reading but
// not its time zone, which Excel does not store.
func (dt *DataTable) ToExcel(path string, opts ExcelWriteOptions) error {
	sheet := opts.sheetName()
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
		case opts.IfSheetExists != SheetExistsReplace:
			return fmt.Errorf("%w: %s in %s", ErrSheetExists, f.GetSheetName(idx), path)
		default:
			if err := replaceSheetInPlace(f, sheet, idx); err != nil {
				return fmt.Errorf("failed to replace sheet %s: %w", sheet, err)
			}
		}
	}

	if err := dt.writeExcelRows(f, sheet, opts); err != nil {
		return err
	}
	return writeFileAtomically(path, func(w io.Writer) error {
		_, err := f.WriteTo(w)
		return err
	})
}

// WriteExcel writes the table as a one-sheet workbook to any destination — an
// HTTP response, a zip entry, a buffer — the same way ToExcel writes a new
// file.
func (dt *DataTable) WriteExcel(w io.Writer, opts ExcelWriteOptions) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := opts.sheetName()
	if err := f.SetSheetName(defaultExcelSheet, sheet); err != nil {
		return fmt.Errorf("failed to name sheet %s: %w", sheet, err)
	}
	if err := dt.writeExcelRows(f, sheet, opts); err != nil {
		return err
	}
	_, err := f.WriteTo(w)
	return err
}

// openOrNewWorkbook opens the workbook at path, or, when nothing is there,
// returns a new one whose only sheet is named sheet and reports it as
// created. Path is recorded on a new workbook so writing it checks the
// extension and sets the matching content type, as opening an existing one
// does.
func openOrNewWorkbook(path, sheet string) (*excelize.File, bool, error) {
	f, err := excelize.OpenFile(path, ExcelReadOptions())
	if err == nil {
		return f, false, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, false, fmt.Errorf("failed to open Excel file %s: %w", path, err)
	}
	f = excelize.NewFile()
	f.Path = path
	if err := f.SetSheetName(defaultExcelSheet, sheet); err != nil {
		_ = f.Close()
		return nil, false, fmt.Errorf("failed to name sheet %s: %w", sheet, err)
	}
	return f, true, nil
}

// replaceSheetInPlace empties the sheet at idx by deleting and re-creating it,
// then moves it back to where it was and restores the active sheet. excelize
// refuses to delete a workbook's only sheet, so that case goes through a
// placeholder. Sheet names match without regard to case, as in Excel.
func replaceSheetInPlace(f *excelize.File, sheet string, idx int) error {
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

// writeExcelRows writes the table into sheet, which must be empty.
func (dt *DataTable) writeExcelRows(f *excelize.File, sheet string, opts ExcelWriteOptions) error {
	var rows [][]any
	dt.AtomicDo(func(dt *DataTable) {
		maxLength := dt.getMaxColLength()
		if opts.SetColNamesToFirstRow {
			header := make([]any, 0, len(dt.columns)+1)
			if opts.SetRowNamesToFirstCol {
				header = append(header, nil)
			}
			for _, column := range dt.columns {
				header = append(header, column.name)
			}
			rows = append(rows, header)
		}
		for rowIndex := 0; rowIndex < maxLength; rowIndex++ {
			record := make([]any, 0, len(dt.columns)+1)
			if opts.SetRowNamesToFirstCol {
				rowName, _ := dt.GetRowNameByIndex(rowIndex)
				record = append(record, rowName)
			}
			for _, column := range dt.columns {
				var value any
				if rowIndex < len(column.data) {
					value = excelCellValue(column.data[rowIndex])
				}
				record = append(record, value)
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
			return fmt.Errorf("failed to write row %d of sheet %s: %w", i+1, sheet, err)
		}
	}
	return nil
}

// excelCellValue passes the values Excel stores natively to excelize and
// writes everything else as the same text ToCSV and ToJSON would.
func excelCellValue(v any) any {
	switch v.(type) {
	case nil, string, bool, time.Time,
		int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return v
	}
	return utils.ValueText(v)
}
