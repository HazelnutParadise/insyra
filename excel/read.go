// Package excel reads Excel workbooks into DataTables and writes DataTables
// as Excel sheets.
//
// From 0.4 it is the only place the core's Excel readers and writers live,
// so a program that only uses the insyra core no longer compiles the Excel
// library into its binary. In 0.3.x the core's ReadExcelSheet, ReadExcel,
// ToExcel and WriteExcel still exist and are deprecated in favour of this
// package. Converting CSV files to workbooks and back without building a
// table is csvxl's job.
//
//	dt, err := excel.Read("sales.xlsx", excel.ReadOptions{Sheet: "2026"})
//	err = excel.Write(dt, "summary.xlsx", excel.WriteOptions{Sheet: "summary"})
//
// The functions take no context: excelize can stop neither opening nor
// decompressing a workbook part-way, so a context could not change when a
// call returns.
package excel

import (
	"bytes"
	"fmt"
	"io"

	"github.com/HazelnutParadise/insyra"
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
// settings ReadOptions describes; at most one may be given.
func Read(path string, opts ...ReadOptions) (*insyra.DataTable, error) {
	opt, err := oneOption("ReadOptions", opts)
	if err != nil {
		return nil, err
	}
	if opt.Sheet == "" {
		sheets, err := SheetNames(path)
		if err != nil {
			return nil, err
		}
		if opt.Sheet, err = firstSheet(sheets); err != nil {
			return nil, err
		}
	}
	return insyra.ReadExcelSheet(path, opt.Sheet, opt.HasRowNames, !opt.NoHeaderRow, opt.RawStrings) //nolint:staticcheck // the deprecated reader is this function's implementation until 0.4
}

// ReadFrom reads one sheet of a workbook from any source — an HTTP response
// body, a zip entry, bytes in memory — the same way Read reads a file. It does
// not close r.
func ReadFrom(r io.Reader, opts ...ReadOptions) (*insyra.DataTable, error) {
	opt, err := oneOption("ReadOptions", opts)
	if err != nil {
		return nil, err
	}
	if opt.Sheet == "" {
		// excelize reads the whole workbook into memory anyway, so holding
		// the bytes to open them twice costs no more than one copy.
		content, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("failed to open Excel input: %w", err)
		}
		f, err := excelize.OpenReader(bytes.NewReader(content))
		if err != nil {
			return nil, fmt.Errorf("failed to open Excel input: %w", err)
		}
		sheets := f.GetSheetList()
		_ = f.Close()
		if opt.Sheet, err = firstSheet(sheets); err != nil {
			return nil, err
		}
		r = bytes.NewReader(content)
	}
	return insyra.ReadExcel(r, opt.Sheet, opt.HasRowNames, !opt.NoHeaderRow, opt.RawStrings) //nolint:staticcheck // the deprecated reader is this function's implementation until 0.4
}

// SheetNames lists the sheets of the workbook at path in the order the
// workbook keeps them, hidden sheets included.
func SheetNames(path string) ([]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return f.GetSheetList(), nil
}

func firstSheet(sheets []string) (string, error) {
	if len(sheets) == 0 {
		return "", fmt.Errorf("the workbook has no sheet")
	}
	return sheets[0], nil
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
