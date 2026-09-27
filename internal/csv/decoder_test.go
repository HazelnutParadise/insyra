package csv

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// Every charset the auto-detector (saintfish/chardet) can report must have a
// decoder, or a file it identifies correctly still fails to load. IBM420,
// IBM424, ISO-2022-KR and ISO-2022-CN are the exceptions: x/text has no
// decoder for them.
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

// v0.3.2 matched a name by substring, so these read there. Each is an alias
// of a charset the table decodes, so it reads here too, in any case and with
// any separators, and utf-8-sig drops the byte-order mark it is named for.
func TestLegacyCharsetAliasesDecode(t *testing.T) {
	const text = "名稱,值\n甲,1\n"
	big5, err := traditionalchinese.Big5.NewEncoder().String(text)
	if err != nil {
		t.Fatal(err)
	}
	gbk, err := simplifiedchinese.GBK.NewEncoder().String(text)
	if err != nil {
		t.Fatal(err)
	}
	decode := func(t *testing.T, data, name string) string {
		t.Helper()
		r, err := DecodingReader(strings.NewReader(data), name)
		if err != nil {
			t.Fatalf("DecodingReader(%q): %v", name, err)
		}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("reading through %q: %v", name, err)
		}
		return string(got)
	}
	for _, c := range []struct {
		data  string
		names []string
	}{
		{big5, []string{"big5-hkscs", "BIG5-HKSCS", "csbig5", "cn-big5", "x-x-big5"}},
		{gbk, []string{"x-gbk", "X-GBK", "gb_2312-80", "csgb2312", "csiso58gb231280", "chinese", "iso-ir-58"}},
	} {
		for _, name := range c.names {
			if got := decode(t, c.data, name); got != text {
				t.Errorf("%s: got %q, want %q", name, got, text)
			}
		}
	}
	for _, name := range []string{"utf-8-sig", "UTF-8-SIG", "utf-8-bom", "utf8_sig"} {
		if got := decode(t, "\uFEFFa,b\n", name); got != "a,b\n" {
			t.Errorf("%s with a byte-order mark: got %q, want %q", name, got, "a,b\n")
		}
		if got := decode(t, "a,b\n", name); got != "a,b\n" {
			t.Errorf("%s without a byte-order mark: got %q, want %q", name, got, "a,b\n")
		}
	}
}
