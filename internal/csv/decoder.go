package csv

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
	"golang.org/x/text/transform"
)

// decoders maps a normalised encoding name to its decoder. A nil value means
// "already UTF-8, read the bytes as they are".
//
// The keys are names with every separator removed (see normalizeEncodingName),
// so "ISO-8859-1", "iso8859_1" and "ISO 8859 1" all land on the same entry.
// Every charset the auto-detector can report is covered, plus the aliases
// people actually type.
var decoders = map[string]encoding.Encoding{
	// UTF-8 and its aliases: nothing to decode.
	"":        nil,
	"utf8":    nil,
	"ascii":   nil,
	"usascii": nil,

	// UTF-16 / UTF-32. UseBOM lets a byte-order mark override the assumed
	// endianness, which is how these files are usually written.
	"utf16":   unicode.UTF16(unicode.LittleEndian, unicode.UseBOM),
	"utf16le": unicode.UTF16(unicode.LittleEndian, unicode.UseBOM),
	"utf16be": unicode.UTF16(unicode.BigEndian, unicode.UseBOM),
	"utf32":   utf32.UTF32(utf32.LittleEndian, utf32.UseBOM),
	"utf32le": utf32.UTF32(utf32.LittleEndian, utf32.UseBOM),
	"utf32be": utf32.UTF32(utf32.BigEndian, utf32.UseBOM),

	// CJK.
	"big5":      traditionalchinese.Big5,
	"gb18030":   simplifiedchinese.GB18030,
	"gbk":       simplifiedchinese.GB18030,
	"gb2312":    simplifiedchinese.GB18030,
	"gb":        simplifiedchinese.GB18030,
	"shiftjis":  japanese.ShiftJIS,
	"sjis":      japanese.ShiftJIS,
	"cp932":     japanese.ShiftJIS,
	"ms932":     japanese.ShiftJIS,
	"eucjp":     japanese.EUCJP,
	"iso2022jp": japanese.ISO2022JP,
	"euckr":     korean.EUCKR,
	"cp949":     korean.EUCKR,
	"ksc5601":   korean.EUCKR,

	// Single-byte Western / Cyrillic / Greek / Hebrew / Arabic / Turkish.
	"iso88591":    charmap.ISO8859_1,
	"latin1":      charmap.ISO8859_1,
	"cp819":       charmap.ISO8859_1,
	"iso88592":    charmap.ISO8859_2,
	"latin2":      charmap.ISO8859_2,
	"iso88593":    charmap.ISO8859_3,
	"iso88594":    charmap.ISO8859_4,
	"iso88595":    charmap.ISO8859_5,
	"iso88596":    charmap.ISO8859_6,
	"iso88597":    charmap.ISO8859_7,
	"iso88598":    charmap.ISO8859_8,
	"iso88598i":   charmap.ISO8859_8I,
	"iso88599":    charmap.Windows1254, // ISO-8859-9 is Windows-1254 minus the C1 additions
	"latin5":      charmap.Windows1254,
	"iso885910":   charmap.ISO8859_10,
	"iso885913":   charmap.ISO8859_13,
	"iso885914":   charmap.ISO8859_14,
	"iso885915":   charmap.ISO8859_15,
	"iso885916":   charmap.ISO8859_16,
	"windows1250": charmap.Windows1250,
	"cp1250":      charmap.Windows1250,
	"windows1251": charmap.Windows1251,
	"cp1251":      charmap.Windows1251,
	"windows1252": charmap.Windows1252,
	"cp1252":      charmap.Windows1252,
	"windows1253": charmap.Windows1253,
	"cp1253":      charmap.Windows1253,
	"windows1254": charmap.Windows1254,
	"cp1254":      charmap.Windows1254,
	"windows1255": charmap.Windows1255,
	"cp1255":      charmap.Windows1255,
	"windows1256": charmap.Windows1256,
	"cp1256":      charmap.Windows1256,
	"windows1257": charmap.Windows1257,
	"cp1257":      charmap.Windows1257,
	"windows1258": charmap.Windows1258,
	"cp1258":      charmap.Windows1258,
	"koi8r":       charmap.KOI8R,
	"koi8u":       charmap.KOI8U,
	"ibm866":      charmap.CodePage866,
	"cp866":       charmap.CodePage866,
	"macintosh":   charmap.Macintosh,
	"macroman":    charmap.Macintosh,
}

// normalizeEncodingName lowercases a charset name and drops the separators
// people and detectors disagree about, so "ISO-8859-1", "iso_8859_1" and
// "ISO 8859 1" all compare equal.
func normalizeEncodingName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch r {
		case '-', '_', ' ', '.', ':':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// SupportedEncodings lists the charset names DecodingReader accepts, sorted,
// for use in error messages and documentation.
func SupportedEncodings() []string {
	out := make([]string, 0, len(decoders))
	for name := range decoders {
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// DecodingReader wraps r so its bytes are decoded from the named encoding into
// UTF-8. An empty name means the bytes are already UTF-8.
//
// An encoding the library cannot decode is an ERROR, not a silent
// pass-through: copying undecoded bytes into a DataTable produces cells that
// are not valid UTF-8 and corrupts every later comparison, join and export,
// with nothing to tell the caller it happened.
func DecodingReader(r io.Reader, encodingName string) (io.Reader, error) {
	key := normalizeEncodingName(encodingName)
	enc, ok := decoders[key]
	if !ok {
		return nil, fmt.Errorf("unsupported encoding %q: insyra can decode %s", encodingName, strings.Join(SupportedEncodings(), ", "))
	}
	if enc == nil {
		return r, nil
	}
	return transform.NewReader(r, enc.NewDecoder()), nil
}
