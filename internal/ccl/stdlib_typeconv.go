package ccl

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// displayedDecimal is a number with its thousands, if any, grouped in threes.
var displayedDecimal = regexp.MustCompile(`^[+-]?(\d{1,3}(,\d{3})+|\d*)(\.\d*)?$`)

// displayedNumber reads text the way a spreadsheet displays a number, with a
// percent sign or thousands commas. displayed is false for text with neither,
// which is left to the ordinary number rule.
func displayedNumber(text string) (value float64, ok bool, displayed bool) {
	body := strings.TrimSpace(text)
	percent := strings.HasSuffix(body, "%")
	if !percent && !strings.Contains(body, ",") {
		return 0, false, false
	}
	exponent := ""
	if percent {
		body = strings.TrimSpace(strings.TrimSuffix(body, "%"))
		exponent = "e-2"
	}
	if !displayedDecimal.MatchString(body) || !strings.ContainsAny(body, "0123456789") {
		return 0, false, true
	}
	// Shifting the exponent, not dividing by 100, keeps 12.3% at the float64
	// closest to 0.123.
	f, err := strconv.ParseFloat(strings.ReplaceAll(body, ",", "")+exponent, 64)
	if err != nil {
		return 0, false, true
	}
	return f, true, true
}

// registerTypeConversionFunctions registers value coercion and null-handling
// helpers (VALUE, TEXT, BOOL, COALESCE, IFNULL).
//
// IFNULL differs from IFNA: IFNA only triggers on float NaN or the string
// "#N/A", while IFNULL replaces actual nil values.
func registerTypeConversionFunctions() {
	tonum := func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("requires 1 argument")
		}
		// Treat nil as nil (caller can wrap with COALESCE if they want a default).
		if args[0] == nil {
			return nil, nil
		}
		if text, isText := args[0].(string); isText {
			if f, ok, displayed := displayedNumber(text); displayed {
				if !ok {
					return nil, nil
				}
				return f, nil
			}
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, nil
		}
		return f, nil
	}
	registerRemovedName("TONUM", "VALUE")
	registerFunction("VALUE", func(args ...any) (any, error) {
		v, err := tonum(args...)
		if err != nil {
			return nil, fmt.Errorf("VALUE %w", err)
		}
		return v, nil
	})

	// TEXT: the 1-arg form converts to string with default formatting;
	// 2-arg form reads the second arg as a Go fmt verb (e.g. "%.2f") or an Excel
	// code (e.g. "0.00", "yyyy-mm-dd"); see text_format.go.
	tostr := func(args ...any) (any, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("requires 1 or 2 arguments")
		}
		if len(args) == 1 {
			return toString(args[0]), nil
		}
		format, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("format arg must be a string, got %T", args[1])
		}
		if kind := kindOfFormat(format); kind != kindGoVerb {
			return textWithFormat(args[0], format, kind)
		}
		out := fmt.Sprintf(format, args[0])
		// fmt writes its own complaint into the string when the verb does not
		// match the value — TEXT(1.5, '%d') gave "%!d(float64=1.5)" — and that
		// went straight into the cell. Report it instead.
		marker := fmtErrorMarker(out, format, fmt.Sprint(args[0]))
		if marker == "" {
			return out, nil
		}
		// An integer under a float verb, TEXT(50, '%.1f'), is formatted as
		// the float64 it equals.
		if i, ok := exactInt(args[0]); ok {
			f := float64(i)
			if out := fmt.Sprintf(format, f); fmtErrorMarker(out, format, fmt.Sprint(f)) == "" {
				return out, nil
			}
		}
		return nil, fmt.Errorf("format %q does not fit %T: %s", format, args[0], marker)
	}
	registerRemovedName("TOSTR", "TEXT")
	registerFunction("TEXT", func(args ...any) (any, error) {
		v, err := tostr(args...)
		if err != nil {
			return nil, fmt.Errorf("TEXT %w", err)
		}
		return v, nil
	})

	registerRemovedName("TOBOOL", "BOOL")
	registerFunction("BOOL", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("BOOL requires 1 argument")
		}
		if args[0] == nil {
			return nil, nil
		}
		b, ok := toBool(args[0])
		if !ok {
			return nil, nil
		}
		return b, nil
	})

	// COALESCE returns the first non-nil, non-NaN argument; otherwise nil.
	registerFunction("COALESCE", func(args ...any) (any, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("COALESCE requires at least 1 argument")
		}
		for _, a := range args {
			if a == nil {
				continue
			}
			if f, ok := a.(float64); ok && math.IsNaN(f) {
				continue
			}
			return a, nil
		}
		return nil, nil
	})

	registerFunction("IFNULL", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("IFNULL requires 2 arguments")
		}
		if args[0] == nil {
			return args[1], nil
		}
		return args[0], nil
	})
}

// fmtErrorMarker reports the first formatting-error marker fmt left in s, or ""
// when there is none. fmt writes these, so their presence means the format and
// the value did not match. A marker that already appears in one of known, the
// format or the value's own text, was put there by the caller rather than by
// fmt and is not reported: TEXT('Item (MISSING)', '%s') is fine.
//
// A bare "(MISSING)" is not a marker: fmt writes it only as "%!<verb>(MISSING)",
// which the verb scan below already finds.
func fmtErrorMarker(s string, known ...string) string {
	fromCaller := func(m string) bool {
		for _, k := range known {
			if strings.Contains(k, m) {
				return true
			}
		}
		return false
	}
	for _, m := range []string{"%!(NOVERB)", "%!(EXTRA ", "%!(BADPREC)", "%!(BADWIDTH)"} {
		if strings.Contains(s, m) && !fromCaller(m) {
			return m
		}
	}
	// %!<verb>( — the shape fmt uses for a verb that does not fit the value.
	// The verb is any rune, a punctuation mark or a non-ASCII letter included:
	// TEXT(1, '%v %_') gets "1 %!_(MISSING)".
	for i := 0; i+2 < len(s); i++ {
		if s[i] != '%' || s[i+1] != '!' {
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i+2:])
		end := i + 2 + size
		if end >= len(s) || s[end] != '(' {
			continue
		}
		if m := s[i : end+1]; !fromCaller(m) {
			return m
		}
	}
	return ""
}
