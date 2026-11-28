// Package excel reads Excel workbooks into DataTables and writes DataTables
// as Excel sheets.
//
// It is a package of its own so that a program that only uses the insyra
// core does not compile the Excel library into its binary. Converting CSV
// files to workbooks and back without building a table is csvxl's job.
//
//	dt, err := excel.Read("sales.xlsx", excel.ReadOptions{Sheet: "2026"})
//	err = excel.Write(dt, "summary.xlsx", excel.WriteOptions{Sheet: "summary"})
//
// The functions take no context: excelize can stop neither opening nor
// decompressing a workbook part-way, so a context could not change when a
// call returns.
package excel

import (
	"fmt"
	"io"

	"github.com/HazelnutParadise/insyra"
	csvInternal "github.com/HazelnutParadise/insyra/internal/csv"
	"github.com/HazelnutParadise/insyra/internal/excelsheet"
	"github.com/xuri/excelize/v2"
)

// ReadOptions configures Read and ReadFrom. The zero value reads the
// workbook's first sheet, takes its first row as the column names, keeps no
// row names and infers each column's type.
type ReadOptions struct {
	// Sheet is the sheet to read. Empty means the first sheet of the workbook.
	Sheet string
	// NoHeaderRow says the sheet has no row of column names, so its first
	// row is data.
	NoHeaderRow bool
	// HasRowNames says the sheet's first column holds row names.
	HasRowNames bool
	// RawStrings keeps every cell as the text Excel displays. By default a
	// column whose every cell reads as a number becomes numbers, the rule CSV
	// reading follows.
	RawStrings bool
}

// Read reads one sheet of the workbook at path into a DataTable. opts are the
// settings ReadOptions describes; at most one may be given. A workbook may
// expand to at most 512 MB while it is read, which keeps a zip bomb from
// exhausting memory.
func Read(path string, opts ...ReadOptions) (*insyra.DataTable, error) {
	opt, err := oneOption("ReadOptions", opts)
	if err != nil {
		return nil, err
	}
	f, err := excelize.OpenFile(path, excelsheet.Options())
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return readSheet(f, opt)
}

// ReadFrom reads one sheet of a workbook from any source — an HTTP response
// body, a zip entry, bytes in memory — the same way Read reads a file, with
// the same limit on how far the workbook may expand. It does not close r.
func ReadFrom(r io.Reader, opts ...ReadOptions) (*insyra.DataTable, error) {
	opt, err := oneOption("ReadOptions", opts)
	if err != nil {
		return nil, err
	}
	f, err := excelize.OpenReader(r, excelsheet.Options())
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel input: %w", err)
	}
	defer func() { _ = f.Close() }()
	return readSheet(f, opt)
}

// SheetNames lists the sheets of the workbook at path in the order the
// workbook keeps them, hidden sheets included.
func SheetNames(path string) ([]string, error) {
	f, err := excelize.OpenFile(path, excelsheet.Options())
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return f.GetSheetList(), nil
}

// readSheet is the sheet reader behind Read and ReadFrom.
func readSheet(f *excelize.File, opt ReadOptions) (*insyra.DataTable, error) {
	sheet := opt.Sheet
	if sheet == "" {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("the workbook has no sheet")
		}
		sheet = sheets[0]
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from sheet %s: %w", sheet, err)
	}
	dt, err := insyra.ReadSlice2D(rows)
	if err != nil {
		return nil, fmt.Errorf("failed when converting sheet %s to DataTable: %w", sheet, err)
	}
	if opt.HasRowNames {
		dt.SetColToRowNames("A")
	}
	if !opt.NoHeaderRow {
		dt.SetRowToColNames(0)
	}
	if !opt.RawStrings {
		inferColumnTypes(dt)
	}
	return dt, nil
}

// inferColumnTypes types each column by the rule CSV reading follows. dt is
// the table readSheet has just built, so nothing else can hold it yet.
func inferColumnTypes(dt *insyra.DataTable) {
	for i := range dt.NumCols() {
		col := dt.GetColByNumber(i)
		// Every cell is a string, a number or nil, so NewDataList has
		// nothing to flatten.
		typed := insyra.NewDataList(csvInternal.InferColumn(col.Data())...)
		dt.UpdateColByNumber(i, typed.SetName(col.GetName()))
	}
}

// oneOption returns the single options value of a trailing ...XxxOptions, or
// the zero value when none was given. More than one is an error, never "first
// wins".
func oneOption[T any](what string, opts []T) (T, error) {
	var zero T
	switch len(opts) {
	case 0:
		return zero, nil
	case 1:
		return opts[0], nil
	}
	return zero, fmt.Errorf("at most one %s may be given, got %d", what, len(opts))
}
