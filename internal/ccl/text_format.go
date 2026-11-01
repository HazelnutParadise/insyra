package ccl

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TEXT and FORMAT_DATE read a format written the Go way or the Excel way, and
// tell which from the format alone:
//
//   - a Go fmt verb, such as %.2f, is Go;
//   - a digit from 1 to 9 is a Go date layout, such as 2006-01-02 or 15:04, since
//     Excel's codes have none;
//   - 0 or # with no letters is an Excel number code, such as 0.00 or #,##0;
//   - anything else with letters is a date format, in Excel's codes or Go's names.

// goVerb matches a Go fmt verb that takes a value.
var goVerb = regexp.MustCompile(`%[-+# 0]*\d*(\.\d+)?[a-zA-Z]`)

// goLayoutDigit is what only a Go date layout holds.
var goLayoutDigit = regexp.MustCompile(`[1-9]`)

// formatKind says how a format reads.
type formatKind int

const (
	kindGoVerb formatKind = iota
	kindDate
	kindExcelNumber
	kindUnknown
)

// kindOfFormat decides how a format reads, without looking at the value.
func kindOfFormat(format string) formatKind {
	switch {
	case goVerb.MatchString(format):
		return kindGoVerb
	case goLayoutDigit.MatchString(format):
		return kindDate
	case hasUnquotedLetter(format):
		return kindDate
	case strings.ContainsAny(unquote(format), "0#"):
		return kindExcelNumber
	}
	return kindUnknown
}

// unquote drops the parts of a format that are literal text: "..." and \x.
func unquote(format string) string {
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(format); i++ {
		switch c := format[i]; {
		case c == '"':
			inQuote = !inQuote
		case inQuote:
		case c == '\\':
			i++
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func hasUnquotedLetter(format string) bool {
	for _, c := range []byte(unquote(format)) {
		if isASCIILetter(c) {
			return true
		}
	}
	return false
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// formatDateValue formats t by a date format of either kind, in t's own zone.
func formatDateValue(t time.Time, format string) (string, error) {
	if goLayoutDigit.MatchString(format) {
		return t.Format(format), nil
	}
	return formatExcelDate(t, format)
}

// dateToken is one piece of a date format: literal text, or a code.
type dateToken struct {
	literal string
	code    string // lower case for an Excel code; a Go name as written
}

// goDateNames are Go layout names that hold no digit. They are matched before
// Excel's codes, case-sensitively, so Mon is not read as m and an error.
var goDateNames = []string{"January", "Monday", "Jan", "Mon", "MST"}

// formatExcelDate formats t by Excel's date codes in any case, with Go's names
// also read. Any other ASCII letter is an error, so a typo cannot come back as
// its own text.
func formatExcelDate(t time.Time, format string) (string, error) {
	tokens, err := scanDateFormat(format)
	if err != nil {
		return "", err
	}
	twelveHour := false
	for _, tok := range tokens {
		if tok.code == "ampm" || tok.code == "ap" {
			twelveHour = true
		}
	}
	var b strings.Builder
	for i, tok := range tokens {
		if tok.code == "" {
			b.WriteString(tok.literal)
			continue
		}
		b.WriteString(dateCodeText(t, tok, tokens, i, twelveHour))
	}
	return b.String(), nil
}

// scanDateFormat splits a date format into its pieces.
func scanDateFormat(format string) ([]dateToken, error) {
	var tokens []dateToken
	lit := func(s string) {
		if n := len(tokens); n > 0 && tokens[n-1].code == "" {
			tokens[n-1].literal += s
			return
		}
		tokens = append(tokens, dateToken{literal: s})
	}
	seenCode := false
scan:
	for i := 0; i < len(format); {
		rest := format[i:]
		switch c := format[i]; {
		case c == '"':
			end := strings.IndexByte(rest[1:], '"')
			if end < 0 {
				return nil, fmt.Errorf("date format %q has an unclosed quote", format)
			}
			lit(rest[1 : end+1])
			i += end + 2
			continue scan
		case c == '\\' && i+1 < len(format):
			lit(format[i+1 : i+2])
			i += 2
			continue scan
		}
		for _, name := range goDateNames {
			if strings.HasPrefix(rest, name) {
				tokens = append(tokens, dateToken{code: name})
				seenCode = true
				i += len(name)
				continue scan
			}
		}
		for _, m := range []struct{ text, code string }{{"AM/PM", "ampm"}, {"A/P", "ap"}} {
			if len(rest) >= len(m.text) && strings.EqualFold(rest[:len(m.text)], m.text) {
				tokens = append(tokens, dateToken{code: m.code, literal: rest[:len(m.text)]})
				seenCode = true
				i += len(m.text)
				continue scan
			}
		}
		if rest[:min(2, len(rest))] == "PM" || rest[:min(2, len(rest))] == "pm" {
			tokens = append(tokens, dateToken{code: "ampm", literal: "AM/PM"})
			seenCode = true
			i += 2
			continue
		}
		c := format[i]
		if c == '.' && len(tokens) > 0 && strings.HasPrefix(tokens[len(tokens)-1].code, "s") {
			n := 1
			for i+n < len(format) && format[i+n] == '0' {
				n++
			}
			if n > 1 && n <= 4 {
				tokens = append(tokens, dateToken{code: "fraction", literal: format[i : i+n]})
				i += n
				continue
			}
		}
		if !isASCIILetter(c) {
			if c == '0' || c == '#' {
				return nil, fmt.Errorf("date format %q holds %q, which is not a date code", format, string(c))
			}
			lit(format[i : i+1])
			i++
			continue
		}
		lower := c | 0x20
		n := 1
		for i+n < len(format) && format[i+n]|0x20 == lower {
			n++
		}
		code := strings.Repeat(string(lower), n)
		switch {
		case lower == 'y' && n <= 4, lower == 'm' && n <= 4, lower == 'd' && n <= 4, lower == 'h' && n <= 2, lower == 's' && n <= 2:
			tokens = append(tokens, dateToken{code: code})
			seenCode = true
			i += n
		default:
			return nil, fmt.Errorf("date format %q holds %q, which is not a date code; use y, m, d, h and s, or a Go layout such as 2006-01-02", format, format[i:i+n])
		}
	}
	if !seenCode {
		return nil, fmt.Errorf("date format %q holds no date code", format)
	}
	return tokens, nil
}

// dateCodeText gives what one code writes. m and mm are minutes right after an
// hour code or right before a second code, as in Excel, and months otherwise.
func dateCodeText(t time.Time, tok dateToken, tokens []dateToken, i int, twelveHour bool) string {
	switch tok.code {
	case "January", "Monday", "Jan", "Mon", "MST":
		return t.Format(tok.code)
	case "y", "yy":
		return fmt.Sprintf("%02d", t.Year()%100)
	case "yyy", "yyyy":
		return strconv.Itoa(t.Year())
	case "m", "mm":
		minutes := t.Minute()
		isMinute := isClockCode(tokens, i, -1, "h") || isClockCode(tokens, i, 1, "s")
		value := int(t.Month())
		if isMinute {
			value = minutes
		}
		if tok.code == "m" {
			return strconv.Itoa(value)
		}
		return fmt.Sprintf("%02d", value)
	case "mmm":
		return t.Format("Jan")
	case "mmmm":
		return t.Format("January")
	case "d":
		return strconv.Itoa(t.Day())
	case "dd":
		return fmt.Sprintf("%02d", t.Day())
	case "ddd":
		return t.Format("Mon")
	case "dddd":
		return t.Format("Monday")
	case "h", "hh":
		hour := t.Hour()
		if twelveHour {
			hour %= 12
			if hour == 0 {
				hour = 12
			}
		}
		if tok.code == "h" {
			return strconv.Itoa(hour)
		}
		return fmt.Sprintf("%02d", hour)
	case "s":
		return strconv.Itoa(t.Second())
	case "ss":
		return fmt.Sprintf("%02d", t.Second())
	case "fraction":
		digits := len(tok.literal) - 1
		return "." + fmt.Sprintf("%09d", t.Nanosecond())[:digits]
	case "ampm":
		upper, lower := "AM", "am"
		if t.Hour() >= 12 {
			upper, lower = "PM", "pm"
		}
		if strings.HasPrefix(tok.literal, "a") {
			return lower
		}
		return upper
	case "ap":
		upper := "A"
		if t.Hour() >= 12 {
			upper = "P"
		}
		if strings.HasPrefix(tok.literal, "a") {
			return strings.ToLower(upper)
		}
		return upper
	}
	return ""
}

// isClockCode reports whether the nearest code from i in direction dir starts
// with prefix, skipping literal text.
func isClockCode(tokens []dateToken, i, dir int, prefix string) bool {
	for j := i + dir; j >= 0 && j < len(tokens); j += dir {
		if tokens[j].code == "" {
			continue
		}
		return strings.HasPrefix(tokens[j].code, prefix)
	}
	return false
}

// numberFormat is an Excel number code taken apart.
type numberFormat struct {
	prefix, suffix string
	minInt         int
	grouped        bool
	minFrac        int
	maxFrac        int
	percents       int
}

// parseNumberFormat reads an Excel number code: 0, #, a comma between digits, a
// point, % and literal text around them. Anything else, such as a ; between
// sections or a scaling comma, is an error naming the format.
func parseNumberFormat(format string) (numberFormat, error) {
	var nf numberFormat
	bad := func(what string) (numberFormat, error) {
		return nf, fmt.Errorf("number format %q: %s is not supported; use 0, #, a thousands comma, a point and %%", format, what)
	}
	var prefix, suffix strings.Builder
	const (
		before = iota
		integer
		fraction
		after
	)
	state := before
	lastComma := false
	literal := func(s string) {
		if state == before {
			prefix.WriteString(s)
		} else {
			state = after
			suffix.WriteString(s)
		}
	}
	for i := 0; i < len(format); i++ {
		c := format[i]
		switch {
		case c == '"':
			end := strings.IndexByte(format[i+1:], '"')
			if end < 0 {
				return nf, fmt.Errorf("number format %q has an unclosed quote", format)
			}
			literal(format[i+1 : i+1+end])
			i += end + 1
		case c == '\\' && i+1 < len(format):
			literal(format[i+1 : i+2])
			i++
		case c == ';':
			return bad("a ; between sections")
		case c == '%':
			nf.percents++
			literal("%")
		case c == '0' || c == '#':
			switch state {
			case before, integer:
				state = integer
				lastComma = false
				if c == '0' {
					nf.minInt++
				} else if nf.minInt > 0 {
					return bad("a # after a 0")
				}
			case fraction:
				if c == '0' {
					if nf.maxFrac > nf.minFrac {
						return bad("a 0 after a # in the decimals")
					}
					nf.minFrac++
				}
				nf.maxFrac++
			case after:
				return bad("digits after text")
			}
		case c == ',':
			if state != integer {
				return bad("a comma outside the digits")
			}
			nf.grouped = true
			lastComma = true
		case c == '.':
			if state == fraction || state == after {
				return bad("a second point")
			}
			state = fraction
		default:
			literal(format[i : i+1])
		}
	}
	if lastComma {
		return bad("a scaling comma")
	}
	nf.prefix, nf.suffix = prefix.String(), suffix.String()
	return nf, nil
}

// formatExcelNumber writes f by an Excel number code. It rounds half away from
// zero on the shortest decimal text of f, so 1.005 under 0.00 is 1.01 as in Excel,
// where Go's %.2f gives 1.00 from the binary value.
func formatExcelNumber(f float64, format string) (string, error) {
	nf, err := parseNumberFormat(format)
	if err != nil {
		return "", err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", fmt.Errorf("number format %q cannot write %v", format, f)
	}
	intPart, fracPart := roundDecimal(math.Abs(f), nf.percents*2, nf.maxFrac)
	fracPart = strings.TrimRight(fracPart, "0")
	for len(fracPart) < nf.minFrac {
		fracPart += "0"
	}
	intPart = strings.TrimLeft(intPart, "0")
	for len(intPart) < nf.minInt {
		intPart = "0" + intPart
	}
	if nf.grouped {
		intPart = groupThousands(intPart)
	}
	body := intPart
	if nf.maxFrac > 0 || strings.Contains(unquote(format), ".") {
		body += "." + fracPart
	}
	sign := ""
	if f < 0 && strings.Trim(intPart+fracPart, "0,") != "" {
		sign = "-"
	}
	return sign + nf.prefix + body + nf.suffix, nil
}

// roundDecimal gives |f|, moved shift places left of the point, rounded half up
// to places decimals, as whole and fractional digits.
func roundDecimal(f float64, shift, places int) (string, string) {
	text := strconv.FormatFloat(f, 'f', -1, 64)
	whole, frac, _ := strings.Cut(text, ".")
	for range shift {
		if frac == "" {
			whole += "0"
			continue
		}
		whole += frac[:1]
		frac = frac[1:]
	}
	if len(frac) <= places {
		return whole, frac
	}
	roundUp := frac[places] >= '5'
	digits := []byte(whole + frac[:places])
	if roundUp {
		i := len(digits) - 1
		for ; i >= 0; i-- {
			if digits[i] != '9' {
				digits[i]++
				break
			}
			digits[i] = '0'
		}
		if i < 0 {
			digits = append([]byte{'1'}, digits...)
		}
	}
	n := len(digits) - places
	return string(digits[:n]), string(digits[n:])
}

// groupThousands puts a comma between each three digits from the right.
func groupThousands(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	var b strings.Builder
	first := len(digits) % 3
	if first > 0 {
		b.WriteString(digits[:first])
	}
	for i := first; i < len(digits); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}

// textWithFormat is TEXT with a format that is not a Go fmt verb. A nil stays
// nil, as VALUE and DUR leave it.
func textWithFormat(val any, format string, kind formatKind) (any, error) {
	if val == nil {
		return nil, nil
	}
	switch kind {
	case kindDate:
		t, ok := toTime(val)
		if !ok {
			return nil, fmt.Errorf("format %q is a date format, and %v (%T) is not a date", format, val, val)
		}
		return formatDateValue(t, format)
	case kindExcelNumber:
		var f float64
		switch x := val.(type) {
		case bool, time.Time:
			return nil, fmt.Errorf("format %q is a number format, and %v (%T) is not a number", format, x, x)
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
			if err != nil {
				return nil, fmt.Errorf("format %q is a number format, and %q is not a number", format, x)
			}
			f = parsed
		default:
			parsed, ok := toFloat64(x)
			if !ok {
				return nil, fmt.Errorf("format %q is a number format, and %T is not a number", format, x)
			}
			f = parsed
		}
		return formatExcelNumber(f, format)
	}
	return nil, fmt.Errorf("format %q is neither a Go format such as %%.2f nor an Excel one such as 0.00 or yyyy-mm-dd", format)
}
