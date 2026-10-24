package isr

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra"
)

// To writes the table to target, the output counterpart of DT.From. target is
// one of CSV, JSON or Excel, given by value or by pointer, and its OutputOpts
// say how to write it.
//
// With a FilePath the table goes to that file. A *CSV or *JSON without a
// FilePath receives the output in memory instead, in CSV.String or JSON.Bytes,
// the fields From reads. Excel always needs a FilePath.
//
// JSON uses column names as keys unless JSON.OutputOpts.UseColIndex is set.
//
// To returns an error rather than recording it on the table: writing ends a
// chain, so there is nothing left to chain on.
func (t *DataTable) To(target any) error {
	if t == nil || t.DataTable == nil {
		return errors.New("isr.DT.To: the table is nil")
	}
	switch v := target.(type) {
	case CSV:
		return t.toCSV(&v, false)
	case *CSV:
		if v == nil {
			return errors.New("isr.DT.To: got a nil *CSV")
		}
		return t.toCSV(v, true)
	case JSON:
		return t.toJSON(&v, false)
	case *JSON:
		if v == nil {
			return errors.New("isr.DT.To: got a nil *JSON")
		}
		return t.toJSON(v, true)
	case Excel:
		return t.toExcel(v)
	case *Excel:
		if v == nil {
			return errors.New("isr.DT.To: got a nil *Excel")
		}
		return t.toExcel(*v)
	default:
		return fmt.Errorf("isr.DT.To: cannot write to %T; use isr.CSV, isr.JSON or isr.Excel", target)
	}
}

func (t *DataTable) toCSV(c *CSV, inMemory bool) error {
	opts := insyra.CSVWriteOptions{
		SetRowNamesToFirstCol: c.OutputOpts.RowNames2FirstCol,
		SetColNamesToFirstRow: c.OutputOpts.ColNames2FirstRow,
		IncludeBOM:            c.OutputOpts.IncludeBOM,
	}
	if c.FilePath != "" {
		if err := t.ToCSVWithOptions(c.FilePath, opts); err != nil {
			return fmt.Errorf("isr.DT.To: %w", err)
		}
		return nil
	}
	if !inMemory {
		return errors.New("isr.DT.To: CSV has no FilePath; pass &isr.CSV{} to receive the text in CSV.String")
	}
	var buf bytes.Buffer
	if err := t.WriteCSV(&buf, opts); err != nil {
		return fmt.Errorf("isr.DT.To: %w", err)
	}
	c.String = buf.String()
	return nil
}

func (t *DataTable) toJSON(j *JSON, inMemory bool) error {
	useColNames := !j.OutputOpts.UseColIndex
	if j.FilePath != "" {
		if err := t.ToJSON(j.FilePath, useColNames); err != nil {
			return fmt.Errorf("isr.DT.To: %w", err)
		}
		return nil
	}
	if !inMemory {
		return errors.New("isr.DT.To: JSON has no FilePath; pass &isr.JSON{} to receive the data in JSON.Bytes")
	}
	var buf bytes.Buffer
	if err := t.WriteJSON(&buf, useColNames); err != nil {
		return fmt.Errorf("isr.DT.To: %w", err)
	}
	j.Bytes = buf.Bytes()
	return nil
}

func (t *DataTable) toExcel(e Excel) error {
	if e.FilePath == "" {
		return errors.New("isr.DT.To: Excel has no FilePath")
	}
	err := t.ToExcel(e.FilePath, insyra.ExcelWriteOptions{
		Sheet:                 e.SheetName,
		SetColNamesToFirstRow: e.OutputOpts.ColNames2FirstRow,
		SetRowNamesToFirstCol: e.OutputOpts.RowNames2FirstCol,
		IfSheetExists:         e.OutputOpts.IfSheetExists,
	})
	if err != nil {
		return fmt.Errorf("isr.DT.To: %w", err)
	}
	return nil
}
