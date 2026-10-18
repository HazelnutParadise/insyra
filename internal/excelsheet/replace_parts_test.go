package excelsheet

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func zipText(t *testing.T, path string) map[string]string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		out[f.Name] = string(b)
	}
	return out
}

// A sheet with a comment and a table, replaced, leaves neither part in the saved
// file, and a new table may take the old table's name.
func TestReplaceRemovesCommentsAndTablesOfTheOldSheet(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in.xlsx")
	f := excelize.NewFile()
	_ = f.SetSheetName("Sheet1", "Data")
	if _, err := f.NewSheet("Other"); err != nil {
		t.Fatal(err)
	}
	for i, v := range []string{"k", "v"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue("Data", cell, v)
	}
	_ = f.SetCellValue("Data", "A2", "a")
	_ = f.SetCellValue("Data", "B2", 1)
	if err := f.AddTable("Data", &excelize.Table{Range: "A1:B2", Name: "Orders"}); err != nil {
		t.Fatal(err)
	}
	if err := f.AddComment("Data", excelize.Comment{Cell: "A2", Author: "x", Paragraph: []excelize.RichTextRun{{Text: "SECRET-NOTE"}}}); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(src); err != nil {
		t.Fatal(err)
	}
	if before := zipText(t, src); !strings.Contains(strings.Join(values(before), ""), "SECRET-NOTE") {
		t.Fatal("setup: the comment is not in the source file")
	}

	g, err := excelize.OpenFile(src)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if err := Replace(g, "Data", 0); err != nil {
		t.Fatal(err)
	}
	// The name of the old table is free again.
	_ = g.SetCellValue("Data", "A1", "k")
	_ = g.SetCellValue("Data", "A2", "z")
	if err := g.AddTable("Data", &excelize.Table{Range: "A1:A2", Name: "Orders"}); err != nil {
		t.Fatalf("a new table cannot take the old table's name: %v", err)
	}
	out := filepath.Join(dir, "out.xlsx")
	if err := g.SaveAs(out); err != nil {
		t.Fatal(err)
	}
	after := zipText(t, out)
	if strings.Contains(strings.Join(values(after), ""), "SECRET-NOTE") {
		t.Error("the old comment text is still in the saved file")
	}
	for name := range after {
		if strings.Contains(name, "comments") || strings.Contains(name, "vmlDrawing") {
			t.Errorf("orphaned part %s kept", name)
		}
	}
	tables := 0
	for name := range after {
		if strings.HasPrefix(name, "xl/tables/table") {
			tables++
		}
	}
	if tables != 1 {
		t.Errorf("saved file holds %d table parts, want only the new one", tables)
	}
	if _, err := excelize.OpenFile(out); err != nil {
		t.Errorf("the saved file does not reopen: %v", err)
	}
}

func values(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
