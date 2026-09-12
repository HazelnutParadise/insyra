package utils

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// A Parquet Binary column arrives as a string holding raw bytes, because a
// DataList cell cannot be a slice. Quoting those bytes and handing them to a
// terminal makes the cell unreadable and, worse, knocks the column out of
// alignment: no width calculation can be right for bytes that are not text.
func TestAStringThatIsNotTextIsShownAsBytes(t *testing.T) {
	raw := string([]byte{0x00, 0xff, 0x41})
	got := FormatValue(raw)

	if got != "00ff41" {
		t.Errorf("FormatValue = %q, want %q", got, "00ff41")
	}
	if !utf8.ValidString(got) {
		t.Errorf("the rendering is itself not valid UTF-8: %q", got)
	}
	// The property that makes the column line up: pure ASCII, so the display
	// width equals the length and no terminal can disagree.
	if runewidth.StringWidth(got) != len(got) {
		t.Errorf("width %d != length %d, so the column cannot be aligned", runewidth.StringWidth(got), len(got))
	}
}

func TestALongByteStringIsTruncatedLikeBytes(t *testing.T) {
	raw := string(append([]byte{0xff}, make([]byte, 30)...))
	got := FormatValue(raw)
	if !strings.Contains(got, "bytes)") {
		t.Errorf("a long byte string was not truncated with a count: %q", got)
	}
	if got != FormatValue([]byte(raw)) {
		t.Errorf("a byte string and the same []byte render differently: %q vs %q", got, FormatValue([]byte(raw)))
	}
}

// A byte sequence that happens to contain 0x0a is not a multi-line string.
func TestByteSequenceWithANewlineByteIsStillBytes(t *testing.T) {
	raw := string([]byte{0xff, 0x0a, 0x41})
	if got := FormatValue(raw); got != "ff0a41" {
		t.Errorf("FormatValue = %q, want %q", got, "ff0a41")
	}
}

func TestTextIsStillText(t *testing.T) {
	for in, want := range map[string]string{
		"A-01":    "'A-01'",
		"中文":      "'中文'",
		"emoji 🙂": "'emoji 🙂'",
		"":        "''",
	} {
		if got := FormatValue(in); got != want {
			t.Errorf("FormatValue(%q) = %q, want %q", in, got, want)
		}
	}
	if got := FormatValue("first\nsecond"); got != "'first...'" {
		t.Errorf("a multi-line string = %q", got)
	}
}
