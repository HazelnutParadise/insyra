package csv

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// DecodingReader wraps r so its bytes are decoded from the named encoding into
// UTF-8. An empty name means the bytes are already UTF-8.
//
// An encoding the library cannot decode is an ERROR, not a silent pass-through:
// copying undecoded bytes into a DataTable produces cells that are not valid
// UTF-8 and corrupts every later comparison, join and export, with nothing to
// tell the caller it happened.
func DecodingReader(r io.Reader, encoding string) (io.Reader, error) {
	enc := strings.ToLower(strings.TrimSpace(encoding))
	switch {
	case enc == "" || strings.Contains(enc, "utf-8") || strings.Contains(enc, "utf8") || enc == "ascii" || enc == "us-ascii":
		return r, nil
	case strings.Contains(enc, "big5"):
		return transform.NewReader(r, traditionalchinese.Big5.NewDecoder()), nil
	case strings.Contains(enc, "gbk") || strings.Contains(enc, "gb18030") || strings.Contains(enc, "gb2312") || strings.Contains(enc, "gb-"):
		return transform.NewReader(r, simplifiedchinese.GB18030.NewDecoder()), nil
	case strings.Contains(enc, "utf-16") || strings.Contains(enc, "utf16"):
		return transform.NewReader(r, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()), nil
	case strings.Contains(enc, "windows-1252") || strings.Contains(enc, "cp1252"):
		return transform.NewReader(r, charmap.Windows1252.NewDecoder()), nil
	case strings.Contains(enc, "iso-8859-1") || strings.Contains(enc, "latin-1") || strings.Contains(enc, "latin1"):
		return transform.NewReader(r, charmap.ISO8859_1.NewDecoder()), nil
	case strings.Contains(enc, "windows-1250") || strings.Contains(enc, "cp1250"):
		return transform.NewReader(r, charmap.Windows1250.NewDecoder()), nil
	case strings.Contains(enc, "windows-1251") || strings.Contains(enc, "cp1251"):
		return transform.NewReader(r, charmap.Windows1251.NewDecoder()), nil
	case strings.Contains(enc, "iso-8859-2"):
		return transform.NewReader(r, charmap.ISO8859_2.NewDecoder()), nil
	case strings.Contains(enc, "iso-8859-15"):
		return transform.NewReader(r, charmap.ISO8859_15.NewDecoder()), nil
	case strings.Contains(enc, "shift") || strings.Contains(enc, "sjis"):
		return transform.NewReader(r, japanese.ShiftJIS.NewDecoder()), nil
	case strings.Contains(enc, "euc-jp") || strings.Contains(enc, "eucjp"):
		return transform.NewReader(r, japanese.EUCJP.NewDecoder()), nil
	case strings.Contains(enc, "euc-kr") || strings.Contains(enc, "euckr"):
		return transform.NewReader(r, korean.EUCKR.NewDecoder()), nil
	default:
		return nil, fmt.Errorf("unsupported encoding %q: pass one insyra can decode (utf-8, utf-16, big5, gbk/gb18030, shift-jis, euc-jp, euc-kr, windows-1250/1251/1252, iso-8859-1/2/15)", encoding)
	}
}
