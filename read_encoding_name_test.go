package insyra

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A name no decoder knows used to hand the bytes back undecoded, so a typo such
// as "big6" read a Big5 file as mojibake and returned no error.
func TestReadCSVRefusesAnEncodingNameItCannotDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.csv")
	if err := os.WriteFile(path, []byte("a,b\n1,2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"big6", "utf-9", "not-a-charset"} {
		_, err := ReadCSV_FileWithOptions(path, CSVReadOptions{Encoding: name})
		if err == nil || !strings.Contains(err.Error(), "unsupported encoding") || !strings.Contains(err.Error(), "big5") {
			t.Errorf("ReadCSV_FileWithOptions(%q): error %v, want one that names the supported encodings", name, err)
		}
		if _, err := ReadCSV(strings.NewReader("a,b\n1,2\n"), CSVReadOptions{Encoding: name}); err == nil {
			t.Errorf("ReadCSV(%q): no error", name)
		}
		var streamErr error
		for _, e := range StreamCSV(strings.NewReader("a,b\n1,2\n"), CSVReadOptions{Encoding: name}, 10) {
			if e != nil {
				streamErr = e
			}
		}
		if streamErr == nil {
			t.Errorf("StreamCSV(%q): no error", name)
		}
	}
}

// Names people type that do decode keep working, and the Windows code page
// names for Big5 and GBK are among them.
func TestReadCSVStillAcceptsTheNamesItDecodes(t *testing.T) {
	big5 := []byte{0xA4, 0xA4, 0xA4, 0xE5, '\n', '1', '\n'} // "中文" then a row, in Big5
	path := filepath.Join(t.TempDir(), "big5.csv")
	if err := os.WriteFile(path, big5, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"big5", "Big5", "cp950", "BIG-5"} {
		dt, err := ReadCSV_FileWithOptions(path, CSVReadOptions{Encoding: name, FirstRowToColNames: true})
		if err != nil {
			t.Errorf("%q: %v", name, err)
			continue
		}
		if got := dt.ColNames(); len(got) != 1 || got[0] != "中文" {
			t.Errorf("%q: columns %v, want [中文]", name, got)
		}
	}
	for _, name := range []string{"", "auto", "utf-8", "UTF8", "ascii", "latin1", "windows-1252"} {
		if _, err := ReadCSV(strings.NewReader("a,b\n1,2\n"), CSVReadOptions{Encoding: name}); err != nil {
			t.Errorf("%q: %v", name, err)
		}
	}
}
