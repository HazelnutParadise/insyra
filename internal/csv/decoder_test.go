package csv

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// Every charset the auto-detector (saintfish/chardet) can report must have a
// decoder, or a file it identifies correctly still fails to load.
func TestEveryDetectableCharsetDecodes(t *testing.T) {
	detectable := []string{
		"UTF-8", "UTF-16BE", "UTF-16LE", "UTF-32BE", "UTF-32LE",
		"Big5", "EUC-JP", "EUC-KR", "GB-18030", "Shift_JIS",
		"ISO-8859-1", "ISO-8859-2", "ISO-8859-5", "ISO-8859-6", "ISO-8859-7",
		"ISO-8859-8", "ISO-8859-8-I", "ISO-8859-9", "KOI8-R",
		"windows-1250", "windows-1251", "windows-1252", "windows-1253",
		"windows-1254", "windows-1255", "windows-1256",
	}
	for _, name := range detectable {
		if _, err := DecodingReader(strings.NewReader("a"), name); err != nil {
			t.Errorf("DecodingReader(%q): %v", name, err)
		}
	}
}

// Names differ in separators and case between detectors, users and docs.
func TestEncodingNameAliases(t *testing.T) {
	for _, name := range []string{"ISO-8859-1", "iso8859_1", "ISO 8859 1", "latin1", "LATIN-1", "cp819"} {
		if _, err := DecodingReader(strings.NewReader("a"), name); err != nil {
			t.Errorf("DecodingReader(%q): %v", name, err)
		}
	}
	for _, name := range []string{"", "utf-8", "UTF8", "ascii"} {
		r, err := DecodingReader(strings.NewReader("plain"), name)
		if err != nil {
			t.Fatalf("DecodingReader(%q): %v", name, err)
		}
		got, _ := io.ReadAll(r)
		if string(got) != "plain" {
			t.Errorf("%q: got %q, want the bytes unchanged", name, got)
		}
	}
}

// Decoding actually happens: latin-1 bytes come out as UTF-8 text.
func TestDecodingProducesUTF8(t *testing.T) {
	r, err := DecodingReader(bytes.NewReader([]byte{'c', 'a', 'f', 0xE9}), "iso-8859-1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "café" {
		t.Fatalf("got %q, want %q", got, "café")
	}
}

// An unknown name is refused, and the message says what is available.
func TestUnknownEncodingIsRefused(t *testing.T) {
	_, err := DecodingReader(strings.NewReader("a"), "klingon-1")
	if err == nil {
		t.Fatal("DecodingReader accepted an unknown encoding")
	}
	if !strings.Contains(err.Error(), "klingon-1") || !strings.Contains(err.Error(), "utf8") {
		t.Fatalf("error should name the input and the supported list: %v", err)
	}
}
