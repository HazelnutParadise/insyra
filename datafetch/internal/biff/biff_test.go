package biff

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "jpx", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestReadsAJPXWorkbook(t *testing.T) {
	wb, err := Read(fixture(t, "stock_val_1_260902.xls"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range wb.Sheets {
		names = append(names, s.Name)
	}
	if len(names) != 4 || names[0] != "TSE Prime" || names[3] != "Tokyo & Nagoya" {
		t.Fatalf("sheets = %v", names)
	}
	prime := wb.Sheets[0]
	if prime.Rows() != 78 {
		t.Errorf("rows = %d, want 78", prime.Rows())
	}
	// Text in both encodings of the shared string table: Japanese (UTF-16) and ASCII.
	if c, ok := prime.Cell(12, 0); !ok || c.Text != "自己計" {
		t.Errorf("A13 = %+v", c)
	}
	if c, ok := prime.Cell(12, 2); !ok || c.Text != "Sales" {
		t.Errorf("C13 = %+v", c)
	}
	// Figures are written as text with separators in these files.
	if c, ok := prime.Cell(12, 8); !ok || c.Text != "5,349,098,396" || c.IsNumber {
		t.Errorf("I13 = %+v", c)
	}
	if _, ok := prime.Cell(0, 5); ok {
		t.Error("an empty cell should not be stored")
	}
}

func TestRefusesWhatIsNotAWorkbook(t *testing.T) {
	good := fixture(t, "stock_val_1_260902.xls")
	cases := map[string][]byte{
		"empty":     nil,
		"text":      []byte("this is not an Excel file at all, just some text of a decent length........................................................................................................................................................................................................................................................................................................................................................................................................................................................................................"),
		"truncated": good[:len(good)/3],
		"header":    good[:512],
	}
	for name, data := range cases {
		if _, err := Read(data); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	// Damaged files give an error or a workbook, never a panic.
	for cut := 0; cut < len(good); cut += 4099 {
		damaged := append([]byte(nil), good...)
		for i := cut; i < cut+64 && i < len(damaged); i++ {
			damaged[i] ^= 0xff
		}
		_, _ = Read(damaged)
	}
}

func TestRKNumbers(t *testing.T) {
	for _, c := range []struct {
		rk   uint32
		want float64
	}{
		{(500 << 2) | 2, 500},
		{(12345 << 2) | 3, 123.45}, // integer 12345 divided by 100
		{0xFFFFFFE6, -7},           // -7 << 2 | 2 as an unsigned word
	} {
		if got := rkValue(c.rk); got != c.want {
			t.Errorf("rkValue(%#x) = %v, want %v", c.rk, got, c.want)
		}
	}
	// A floating-point RK holds the top 30 bits of a double: 1.5 is 0x3FF8000000000000.
	if got := rkValue(0x3FF80000); got != 1.5 {
		t.Errorf("rkValue of a double = %v, want 1.5", got)
	}
}
