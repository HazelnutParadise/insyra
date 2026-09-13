package csv

import (
	"bytes"
	"io"
	"testing"
)

func decodeAll(t *testing.T, input []byte, name string) string {
	t.Helper()
	got, err := io.ReadAll(DecodingReader(bytes.NewReader(input), name))
	if err != nil {
		t.Fatalf("%q: %v", name, err)
	}
	return string(got)
}

// Every charset the auto-detector (saintfish/chardet) can report must have a
// decoder, or a file it identifies correctly still loads as raw bytes.
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
		if _, ok := decoders[normalizeEncodingName(name)]; !ok {
			t.Errorf("no decoder for detectable charset %q", name)
		}
	}
}

// Names differ in separators and case between detectors, users and docs.
func TestEncodingNameAliases(t *testing.T) {
	for _, name := range []string{"ISO-8859-1", "iso8859_1", "ISO 8859 1", "latin1", "LATIN-1", "cp819"} {
		if got := decodeAll(t, []byte{'c', 'a', 'f', 0xE9}, name); got != "café" {
			t.Errorf("%q: got %q, want %q", name, got, "café")
		}
	}
	for _, name := range []string{"", "utf-8", "UTF8", "ascii"} {
		if got := decodeAll(t, []byte("plain"), name); got != "plain" {
			t.Errorf("%q: got %q, want the bytes unchanged", name, got)
		}
	}
}

// UTF-16BE without a byte-order mark used to be read as little-endian.
func TestUTF16BEWithoutBOM(t *testing.T) {
	if got := decodeAll(t, []byte{0, 'h', 0, 'i'}, "utf-16be"); got != "hi" {
		t.Fatalf("got %q, want %q", got, "hi")
	}
}

// A name the table does not know keeps the substring rules earlier releases
// used, and anything else passes through undecoded.
func TestUnknownEncodingKeepsLegacyBehaviour(t *testing.T) {
	big5 := []byte{0xA4, 0xA4} // 中
	if got := decodeAll(t, big5, "big5-hkscs"); got != "中" {
		t.Errorf("big5-hkscs: got %q, want %q", got, "中")
	}
	gbk := []byte{0xD6, 0xD0} // 中
	if got := decodeAll(t, gbk, "x-gbk"); got != "中" {
		t.Errorf("x-gbk: got %q, want %q", got, "中")
	}
	if got := decodeAll(t, []byte{0xEF, 0xBB, 0xBF, 'a'}, "utf-8-sig"); got != "\xEF\xBB\xBFa" {
		t.Errorf("utf-8-sig: got %q, want the bytes unchanged", got)
	}
	raw := []byte{'c', 'a', 'f', 0xE9}
	if got := decodeAll(t, raw, "klingon-1"); got != string(raw) {
		t.Errorf("klingon-1: got %q, want the bytes unchanged", got)
	}
}
