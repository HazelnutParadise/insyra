package ccl

import (
	"fmt"
	"math"
	"strings"
)

// registerTypeConversionFunctions registers value coercion and null-handling
// helpers (TONUM/VALUE, TOSTR/TEXT, TOBOOL, COALESCE, IFNULL).
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
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, nil
		}
		return f, nil
	}
	registerFunction("TONUM", func(args ...any) (any, error) {
		v, err := tonum(args...)
		if err != nil {
			return nil, fmt.Errorf("TONUM %v", err)
		}
		return v, nil
	})
	registerFunction("VALUE", func(args ...any) (any, error) {
		v, err := tonum(args...)
		if err != nil {
			return nil, fmt.Errorf("VALUE %v", err)
		}
		return v, nil
	})

	// TOSTR / TEXT: 1-arg forms convert to string with default formatting;
	// 2-arg form treats the second arg as a Go fmt verb format (e.g. "%.2f").
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
		out := fmt.Sprintf(format, args[0])
		// fmt writes its own complaint into the string when the verb does not
		// match the value — TOSTR(1.5, '%d') gave "%!d(float64=1.5)" — and that
		// went straight into the cell. Report it instead.
		if marker := fmtErrorMarker(out); marker != "" {
			return nil, fmt.Errorf("format %q does not fit %T: %s", format, args[0], marker)
		}
		return out, nil
	}
	registerFunction("TOSTR", func(args ...any) (any, error) {
		v, err := tostr(args...)
		if err != nil {
			return nil, fmt.Errorf("TOSTR %v", err)
		}
		return v, nil
	})
	registerFunction("TEXT", func(args ...any) (any, error) {
		v, err := tostr(args...)
		if err != nil {
			return nil, fmt.Errorf("TEXT %v", err)
		}
		return v, nil
	})

	registerFunction("TOBOOL", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("TOBOOL requires 1 argument")
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
// when there is none. fmt writes these and nothing else does, so their presence
// means the format and the value did not match. A value that itself contains a
// marker-shaped substring would be a false positive; that is far rarer than
// writing Go's error text into a column.
func fmtErrorMarker(s string) string {
	for _, m := range []string{"%!(NOVERB)", "%!(EXTRA ", "(MISSING)", "%!(BADPREC)", "%!(BADWIDTH)"} {
		if strings.Contains(s, m) {
			return m
		}
	}
	// %!<verb>( — the shape fmt uses for a verb that does not fit the value.
	for i := 0; i+3 < len(s); i++ {
		if s[i] == '%' && s[i+1] == '!' && s[i+3] == '(' {
			c := s[i+2]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				return s[i : i+4]
			}
		}
	}
	return ""
}
