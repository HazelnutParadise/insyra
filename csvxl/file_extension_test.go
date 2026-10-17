package csvxl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// On the way in the path is read as given when it exists, and .csv is added
// only when nothing is there.

func firstCell(t *testing.T, dir, source string) string {
	t.Helper()
	out := filepath.Join(dir, "out.xlsx")
	_ = os.Remove(out)
	require.NoError(t, CSVToExcel([]string{source}, []string{"s"}, out, UTF8))
	return sheetRows(t, out, "s")[0][0]
}

func TestCSVToExcelReadsANameThatDoesNotEndInCsv(t *testing.T) {
	for _, name := range []string{"export.txt", "DATA.CSV", "plain"} {
		dir := t.TempDir()
		writeCSV(t, dir, name, name+"\n1\n")
		require.Equal(t, name, firstCell(t, dir, filepath.Join(dir, name)), name)
	}
}

func TestCSVToExcelAddsCsvWhenThePathIsNotThere(t *testing.T) {
	for _, c := range []struct{ given, onDisk string }{
		{"data", "data.csv"},
		{"export.txt", "export.txt.csv"},
	} {
		dir := t.TempDir()
		writeCSV(t, dir, c.onDisk, c.onDisk+"\n1\n")
		require.Equal(t, c.onDisk, firstCell(t, dir, filepath.Join(dir, c.given)), c.given)
	}
}

func TestCSVToExcelReadsThePathAsWrittenWhenBothExist(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "x", "literal\n1\n")
	writeCSV(t, dir, "x.csv", "added\n1\n")
	require.Equal(t, "literal", firstCell(t, dir, filepath.Join(dir, "x")))
}

func TestCSVToExcelSkipsADirectoryOfTheSameName(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "data"), 0o755))
	writeCSV(t, dir, "data.csv", "file\n1\n")
	require.Equal(t, "file", firstCell(t, dir, filepath.Join(dir, "data")))
}

func TestAppendCSVToExcelReadsThePathAsWritten(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "export.txt", "cell\n1\n")
	out := filepath.Join(dir, "out.xlsx")
	require.NoError(t, CSVToExcel(nil, nil, out))
	require.NoError(t, AppendCSVToExcel([]string{filepath.Join(dir, "export.txt")}, []string{"s"}, out, UTF8))
	require.Equal(t, "cell", sheetRows(t, out, "s")[0][0])
}
