package insyra

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/traditionalchinese"
)

// writeEncoded writes text in enc to a new file and returns its path.
func writeEncoded(t *testing.T, name, text string, enc encoding.Encoding) string {
	t.Helper()
	b, err := enc.NewEncoder().String(text)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(b), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func numericRows(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "%d,%d.5,%d\n", i, i*3, i%7)
	}
	return b.String()
}

// The detector ties Big5, GB-18030 and Shift_JIS on a file this short, and
// used to pick among them by goroutine order.
func TestAShortBig5FileIsRefusedEveryTime(t *testing.T) {
	p := writeEncoded(t, "short.csv", "名稱,數量\n蘋果,3\n香蕉,5\n", traditionalchinese.Big5)

	for i := 0; i < 50; i++ {
		got, err := DetectEncoding(p)
		if !errors.Is(err, ErrEncodingAmbiguous) {
			t.Fatalf("call %d: got %q, %v; want ErrEncodingAmbiguous", i, got, err)
		}
		if got != "" {
			t.Fatalf("call %d: returned %q along with the error", i, got)
		}
		for _, candidate := range []string{"big5", "gb-18030", "shift_jis"} {
			if !strings.Contains(err.Error(), candidate) {
				t.Fatalf("error does not name %s: %v", candidate, err)
			}
		}
	}
}

// The commonest real file: a Chinese header over numbers.
func TestAChineseHeaderOverNumbersIsRefused(t *testing.T) {
	p := writeEncoded(t, "prices.csv", "名稱,數量,價格\n"+numericRows(400), traditionalchinese.Big5)

	_, err := ReadCSVFile(p)
	if !errors.Is(err, ErrEncodingAmbiguous) {
		t.Fatalf("got %v, want ErrEncodingAmbiguous", err)
	}
	if !strings.Contains(err.Error(), "name the encoding") {
		t.Fatalf("error does not say how to read the file: %v", err)
	}

	dt, err := ReadCSVFile(p, CSVReadOptions{Encoding: "big5"})
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.ColNames(); got[0] != "名稱" {
		t.Fatalf("with the encoding named, the header reads %v", got)
	}
}

func TestAWeakGuessIsRefused(t *testing.T) {
	p := writeEncoded(t, "cafe.csv", "name,city\ncafé,Zürich\n", charmap.ISO8859_1)

	if _, err := DetectEncoding(p); !errors.Is(err, ErrEncodingAmbiguous) {
		t.Fatalf("got %v, want ErrEncodingAmbiguous for a guess below the threshold", err)
	}
}

func TestAConclusiveGuessIsStillMade(t *testing.T) {
	cases := []struct {
		name, text, want string
		enc              encoding.Encoding
	}{
		{"big5.csv", strings.Repeat("台北市,大安區,忠孝東路\n", 10), "big5", traditionalchinese.Big5},
		{"latin1.csv", strings.Repeat("Größe,Straße,Müller,Übung,schön\n", 30), "iso-8859-1", charmap.Windows1252},
	}
	for _, c := range cases {
		p := writeEncoded(t, c.name, c.text, c.enc)
		for i := 0; i < 20; i++ {
			got, err := DetectEncoding(p)
			if err != nil || got != c.want {
				t.Fatalf("%s call %d: got %q, %v; want %q", c.name, i, got, err, c.want)
			}
		}
	}
}

func TestAReaderInputIsJudgedTheSameWay(t *testing.T) {
	b, err := traditionalchinese.Big5.NewEncoder().String("名稱,數量,價格\n" + numericRows(20))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCSV(strings.NewReader(b)); !errors.Is(err, ErrEncodingAmbiguous) {
		t.Fatalf("got %v, want ErrEncodingAmbiguous", err)
	}
}
