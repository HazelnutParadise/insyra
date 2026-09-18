package csvxl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// Docs/csvxl.md has said since v0.3.2 that ReadCsvToString "returns UTF-8
// content". Given an encoding nothing here decodes, it returned the file's raw
// bytes with a nil error.
func TestReadCsvToStringRefusesAnEncodingItCannotDecode(t *testing.T) {
	path := writeCSVBytes(t, []byte("a,b\n1,2\n"))
	got, err := ReadCsvToString(path, "klingon-1")
	if err == nil {
		t.Fatalf("ReadCsvToString(klingon-1) = %q with no error", got)
	}
	if !strings.Contains(err.Error(), "klingon-1") || !strings.Contains(err.Error(), "utf8") {
		t.Fatalf("the error should name the encoding and what is supported: %v", err)
	}
}

// ibm424Trigrams is the detector's IBM424 (EBCDIC Hebrew, right to left)
// trigram table, from saintfish/chardet.
var ibm424Trigrams = []uint32{
	0x404146, 0x404148, 0x404151, 0x404171, 0x404251, 0x404256, 0x404541, 0x404546, 0x404551, 0x404556, 0x404562, 0x404569, 0x404571, 0x405441, 0x405445, 0x405641,
	0x406254, 0x406954, 0x417140, 0x454041, 0x454042, 0x454045, 0x454054, 0x454056, 0x454069, 0x454641, 0x464140, 0x465540, 0x465740, 0x466840, 0x467140, 0x514045,
	0x514540, 0x514671, 0x515155, 0x515540, 0x515740, 0x516840, 0x517140, 0x544041, 0x544045, 0x544140, 0x544540, 0x554041, 0x554042, 0x554045, 0x554054, 0x554056,
	0x554069, 0x564540, 0x574045, 0x584540, 0x585140, 0x585155, 0x625440, 0x684045, 0x685155, 0x695440, 0x714041, 0x714042, 0x714045, 0x714054, 0x714056, 0x714069,
}

// The same holds for an encoding Auto detects but cannot decode. The sample is
// built from the detector's own IBM424 trigrams; the 0xFF bytes keep it from
// passing as UTF-8, which DetectEncoding checks first.
func TestReadCsvToStringRefusesADetectedEncodingItCannotDecode(t *testing.T) {
	var sample []byte
	for range 20 {
		for i, n := range ibm424Trigrams {
			sample = append(sample, byte(n>>16), byte(n>>8), byte(n))
			if i%16 == 15 {
				sample = append(sample, 0xFF)
			}
		}
	}
	path := writeCSVBytes(t, sample)
	detected, err := insyra.DetectEncoding(path)
	if err != nil || detected != "ibm424_rtl" {
		t.Fatalf("the sample should be detected as ibm424_rtl, got %q (%v)", detected, err)
	}
	got, err := ReadCsvToString(path)
	if err == nil {
		t.Fatalf("ReadCsvToString(auto) returned %d bytes with no error", len(got))
	}
	if !strings.Contains(err.Error(), "ibm424_rtl") {
		t.Fatalf("the error should name the detected encoding: %v", err)
	}
}

// A name the decoder table or the older substring rules know reads as before.
func TestReadCsvToStringStillReadsKnownNames(t *testing.T) {
	const text = "名稱,值\n甲,1\n"
	big5, err := traditionalchinese.Big5.NewEncoder().String(text)
	if err != nil {
		t.Fatal(err)
	}
	big5Path := writeCSVBytes(t, []byte(big5))
	for _, name := range []string{"big5", "big5-hkscs"} {
		got, err := ReadCsvToString(big5Path, name)
		if err != nil || got != text {
			t.Errorf("ReadCsvToString(%s) = %q, %v; want %q", name, got, err, text)
		}
	}
	plainPath := writeCSVBytes(t, []byte("a,b\n1,2\n"))
	for _, name := range []string{"utf-8", "utf-8-sig", "ASCII"} {
		got, err := ReadCsvToString(plainPath, name)
		if err != nil || got != "a,b\n1,2\n" {
			t.Errorf("ReadCsvToString(%s) = %q, %v", name, got, err)
		}
	}
}

// Docs/csvxl.md says separators and case do not matter in an encoding name.
// The substring aliases the table does not hold were matched on the name as
// typed, so an upper-case spelling of one failed where its lower-case twin
// read. insyra.ReadCSV_File lower-cases the name before it asks, so the two
// readers disagreed about the same file.
func TestReadCsvToStringReadsMixedCaseAliases(t *testing.T) {
	const text = "名稱,值\n甲,1\n"
	big5, err := traditionalchinese.Big5.NewEncoder().String(text)
	if err != nil {
		t.Fatal(err)
	}
	gbk, err := simplifiedchinese.GB18030.NewEncoder().String(text)
	if err != nil {
		t.Fatal(err)
	}
	plain := "a,b\n1,2\n"

	for _, c := range []struct {
		name string
		data string
		want string
	}{
		{"UTF-8-SIG", plain, plain},
		{"Utf-8-Sig", plain, plain},
		{"BIG5-HKSCS", big5, text},
		{"Big5-HKSCS", big5, text},
		{"X-GBK", gbk, text},
		{"x-GBK", gbk, text},
	} {
		path := writeCSVBytes(t, []byte(c.data))
		got, err := ReadCsvToString(path, c.name)
		if err != nil {
			t.Errorf("ReadCsvToString(%s): %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("ReadCsvToString(%s) = %q, want %q", c.name, got, c.want)
		}
	}

	// A name no rule knows still fails, in any case.
	for _, name := range []string{"klingon-1", "KLINGON-1"} {
		path := writeCSVBytes(t, []byte(plain))
		if got, err := ReadCsvToString(path, name); err == nil {
			t.Errorf("ReadCsvToString(%s) = %q with no error", name, got)
		}
	}
}

// The unsupported-encoding message has to name the aliases that work, not
// only the table's normalised keys: someone told "insyra can decode ... big5"
// after typing "big5-hkscs" has no way to tell that the name they typed would
// have worked.
func TestUnsupportedEncodingErrorNamesTheAliases(t *testing.T) {
	path := writeCSVBytes(t, []byte("a,b\n1,2\n"))
	_, err := ReadCsvToString(path, "klingon-1")
	if err == nil {
		t.Fatal("klingon-1 must be refused")
	}
	for _, want := range []string{"big5", "gb", "utf8", "utf16"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should mention the %q family: %v", want, err)
		}
	}
	if !strings.Contains(err.Error(), "contain") {
		t.Errorf("the error should say a name containing one of those families is accepted: %v", err)
	}
}

func writeCSVBytes(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "in.csv")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
