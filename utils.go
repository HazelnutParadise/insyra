package insyra

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/saintfish/chardet"
	"golang.org/x/term"
)

// ToFloat64 converts a numeric value — any int, uint or float width, a named
// type over one, or a decimal — to float64. A value it cannot convert gives
// 0, which cannot be told apart from a real zero; use ToFloat64Safe when that
// matters.
func ToFloat64(v any) float64 {
	return utils.ToFloat64(v)
}

// ToFloat64Safe converts a numeric value to float64 like ToFloat64, and
// reports false instead of returning a made-up 0 when v is not a number.
func ToFloat64Safe(v any) (float64, bool) {
	return utils.ToFloat64Safe(v)
}

// SliceToF64 converts a []any to a []float64. Every numeric element (all int,
// uint and float widths) is converted to its float64 value; genuinely
// non-numeric elements are substituted with 0 to preserve the slice length.
func SliceToF64(input []any) []float64 {
	out := make([]float64, 0, len(input))
	for _, v := range input {
		if f, ok := ToFloat64Safe(v); ok {
			out = append(out, f)
		} else {
			out = append(out, 0) // Substitute 0 for non-numeric values
		}
	}
	return out
}

// ProcessData reads input as a []any: a slice or an array element by element,
// an IDataList by its values, and a pointer to a slice or an array by what it
// points to. An empty slice gives an empty, non-nil result.
//
// Any other type, a nil input and a nil pointer are an error, with a nil
// result.
func ProcessData(input any) ([]any, error) {
	if input == nil {
		return nil, errors.New("input is nil")
	}
	value := reflect.ValueOf(input)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, fmt.Errorf("input is a nil %T", input)
	}
	if dl, ok := input.(IDataList); ok {
		return dl.Data(), nil
	}
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		data := make([]any, value.Len())
		for i := range value.Len() {
			data[i] = value.Index(i).Interface()
		}
		return data, nil
	}
	return nil, fmt.Errorf("cannot read %T: want a slice, an array or an IDataList", input)
}

// isNumeric checks if the value is numeric.
func IsNumeric(v any) bool {
	if v == nil {
		return false
	}

	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}

	// 處理反射類型
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}

	// A fixed-point decimal is a number. This has to agree with the read
	// path: fix-clear-defects-core fixed exactly this split for named numeric
	// types, and reopening it would leave a value that is a number to one
	// half of the library and unreadable to the other.
	if _, ok := utils.DecimalToFloat64(v); ok {
		return true
	}

	return false
}

// ColIndexToNumber turns an Excel-style column index ("A", "Z", "AA", in
// either case) into its column number, which counts from 0: "A" is 0 and "AA"
// is 26. It reports -1, false for an empty string or any character that is not
// a letter. ColNumberToIndex is the reverse.
func ColIndexToNumber(colIndex string) (colNumber int, ok bool) {
	return utils.ParseColIndex(colIndex)
}

// ColNumberToIndex turns a column number, which counts from 0, into its
// Excel-style column index: 0 is "A" and 26 is "AA". It reports "", false for
// a negative number. ColIndexToNumber is the reverse.
func ColNumberToIndex(colNumber int) (colIndex string, ok bool) {
	return utils.CalcColIndex(colNumber)
}

// colorText wraps text in the ANSI colour code when it is going to w and w
// should be coloured (see useColorFor); otherwise it returns the plain text.
func colorText(w io.Writer, code string, text any) string {
	if useColorFor(w) {
		return fmt.Sprintf("\033[%sm%v\033[0m", code, text)
	}
	return fmt.Sprintf("%v", text)
}

// useColorFor reports whether output to w is coloured: coloured output is on
// (Config.SetUseColoredOutput), the environment allows colour (NO_COLOR is
// unset and TERM is not "dumb"), and w is a terminal. A file, a pipe or a
// buffer gets plain text, so colour codes never end up inside saved output.
func useColorFor(w io.Writer) bool {
	if !Config.GetUseColoredOutput() || !utils.IsColorSupported() {
		return false
	}
	f, ok := w.(*os.File)
	return ok && isTerminal(f)
}

// isTerminal reports whether f is a terminal. It is a variable so tests can
// check the coloured path without one.
var isTerminal = func(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// DetectEncoding detects the character encoding of a text file.
// It reads a sample of the file and returns the detected charset in lower case
// (e.g. "utf-8", "big5", "gb-18030", "utf-16le").
// For empty files it returns an error.
func DetectEncoding(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	// Read a reasonably large sample to improve detection accuracy
	buf := make([]byte, encodingSampleSize)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return detectEncodingOfSample(buf[:n], filePath)
}

// encodingSampleSize is how much of an input the encoding is judged from.
const encodingSampleSize = 8192

// detectEncodingOfSample names the encoding of the first bytes of an input.
// source only labels the messages, so a file and a reader are judged by the
// same rule.
func detectEncodingOfSample(sample []byte, source string) (string, error) {
	if len(sample) == 0 {
		return "", fmt.Errorf("empty file %s", source)
	}

	// BOM checks. UTF-32's BOMs start with UTF-16's, so they must be tested
	// first or every UTF-32LE file is reported as UTF-16LE.
	if bytes.HasPrefix(sample, []byte{0xEF, 0xBB, 0xBF}) {
		return "utf-8", nil
	}
	if bytes.HasPrefix(sample, []byte{0xFF, 0xFE, 0x00, 0x00}) {
		return "utf-32le", nil
	}
	if bytes.HasPrefix(sample, []byte{0x00, 0x00, 0xFE, 0xFF}) {
		return "utf-32be", nil
	}
	if bytes.HasPrefix(sample, []byte{0xFF, 0xFE}) {
		return "utf-16le", nil
	}
	if bytes.HasPrefix(sample, []byte{0xFE, 0xFF}) {
		return "utf-16be", nil
	}

	// Quick UTF-8 heuristic. A sample that filled the buffer may end in the
	// middle of a multi-byte rune, which is not invalid UTF-8, just truncated:
	// back off to the last complete rune before judging. A shorter sample is
	// the whole input, so its tail is judged as it is.
	if utf8.Valid(sample) || (len(sample) == encodingSampleSize && utf8.Valid(trimIncompleteRune(sample))) {
		return "utf-8", nil
	}

	// Fallback to chardet. When it cannot name a charset — which happens on a
	// very short sample — assume UTF-8 rather than failing the whole read; the
	// decoder reports any byte it then cannot handle.
	results, err := chardet.NewTextDetector().DetectAll(sample)
	if err != nil {
		LogWarning("insyra", "DetectEncoding", "could not identify the encoding of %s (%v); assuming utf-8", source, err)
		return "utf-8", nil
	}
	return conclusiveCharset(results, source)
}

// minEncodingConfidence is the lowest chardet confidence (0-100) insyra acts
// on. Measured on 2026-10-13: Big5, GBK, Shift_JIS and EUC-KR files whose
// only non-ASCII text is a short header all scored 10, tied with other CJK
// charsets, and a Shift_JIS file of ten short rows scored 49 as windows-1252;
// Big5, GBK and EUC-KR files with ten rows of text scored 77 to 100.
const minEncodingConfidence = 50

// ErrEncodingAmbiguous is returned, wrapped, when an encoding was to be
// detected and the input does not show which one it is: the best guess is
// below the confidence insyra acts on, or another charset is just as likely.
// Name the encoding to read such an input.
var ErrEncodingAmbiguous = errors.New("cannot tell the encoding")

// conclusiveCharset returns the best of chardet's results when it is clear:
// confident enough and not tied. chardet runs its recognisers concurrently and
// sorts their results unstably, so a tie used to be broken by whichever
// goroutine finished first; a tie is now refused, which keeps the answer the
// same on every call.
func conclusiveCharset(results []chardet.Result, source string) (string, error) {
	sorted := append([]chardet.Result(nil), results...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Confidence != sorted[j].Confidence {
			return sorted[i].Confidence > sorted[j].Confidence
		}
		return strings.ToLower(sorted[i].Charset) < strings.ToLower(sorted[j].Charset)
	})
	best := sorted[0]
	tied := len(sorted) > 1 && sorted[1].Confidence == best.Confidence
	if best.Confidence >= minEncodingConfidence && !tied {
		return strings.ToLower(best.Charset), nil
	}
	var candidates []string
	for i, r := range sorted {
		if i == 5 {
			break
		}
		candidates = append(candidates, fmt.Sprintf("%s (%d)", strings.ToLower(r.Charset), r.Confidence))
	}
	return "", fmt.Errorf("%w of %s: it could be %s, out of 100, and insyra guesses only at %d or more with no tie; name the encoding instead of detecting it",
		ErrEncodingAmbiguous, source, strings.Join(candidates, ", "), minEncodingConfidence)
}

// trimIncompleteRune drops a trailing partial UTF-8 sequence so a sample cut at
// an arbitrary byte boundary can still be judged. Only a valid-but-incomplete
// prefix of a rune is dropped: the bytes from the last rune start must be too
// few for the rune that start announces. Any other tail is returned unchanged.
func trimIncompleteRune(b []byte) []byte {
	for i := len(b) - 1; i >= 0 && i > len(b)-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if !utf8.FullRune(b[i:]) {
				return b[:i]
			}
			return b
		}
	}
	return b
}
