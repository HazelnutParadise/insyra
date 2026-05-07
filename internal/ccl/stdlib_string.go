package ccl

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// toString converts a value to a string for use in CCL string functions.
// nil becomes the empty string. Other scalars use fmt.Sprint to mirror CONCAT.
func toString(val any) string {
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprint(val)
}

// runeAt returns a substring [start, start+length) measured in runes.
// Behaves like Excel: clamps to bounds; returns "" if start is past end.
// start is 0-based here (callers convert from 1-based MID/SUBSTR).
func runeSlice(s string, start, length int) string {
	if length <= 0 {
		return ""
	}
	runes := []rune(s)
	n := len(runes)
	if start < 0 {
		start = 0
	}
	if start >= n {
		return ""
	}
	end := start + length
	if end > n {
		end = n
	}
	return string(runes[start:end])
}

// registerStringFunctions registers Excel/SQL-style scalar string functions.
// All functions are rune-aware; LEN counts runes, LEFT/RIGHT/MID slice by rune.
func registerStringFunctions() {
	registerFunction("LEN", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("LEN requires 1 argument")
		}
		return float64(utf8.RuneCountInString(toString(args[0]))), nil
	})

	registerFunction("UPPER", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("UPPER requires 1 argument")
		}
		return strings.ToUpper(toString(args[0])), nil
	})

	registerFunction("LOWER", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("LOWER requires 1 argument")
		}
		return strings.ToLower(toString(args[0])), nil
	})

	registerFunction("TRIM", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("TRIM requires 1 argument")
		}
		return strings.TrimSpace(toString(args[0])), nil
	})

	registerFunction("LTRIM", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("LTRIM requires 1 argument")
		}
		return strings.TrimLeft(toString(args[0]), " \t\n\r"), nil
	})

	registerFunction("RTRIM", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RTRIM requires 1 argument")
		}
		return strings.TrimRight(toString(args[0]), " \t\n\r"), nil
	})

	registerFunction("LEFT", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("LEFT requires 2 arguments")
		}
		s := toString(args[0])
		n, ok := toFloat64(args[1])
		if !ok {
			return nil, fmt.Errorf("LEFT: count arg must be a number, got %T", args[1])
		}
		return runeSlice(s, 0, int(n)), nil
	})

	registerFunction("RIGHT", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("RIGHT requires 2 arguments")
		}
		s := toString(args[0])
		n, ok := toFloat64(args[1])
		if !ok {
			return nil, fmt.Errorf("RIGHT: count arg must be a number, got %T", args[1])
		}
		count := int(n)
		if count <= 0 {
			return "", nil
		}
		runes := []rune(s)
		if count >= len(runes) {
			return s, nil
		}
		return string(runes[len(runes)-count:]), nil
	})

	// MID(s, start, length) — start is 1-based, matching Excel.
	mid := func(args ...any) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("requires 3 arguments")
		}
		s := toString(args[0])
		start, ok := toFloat64(args[1])
		if !ok {
			return nil, fmt.Errorf("start arg must be a number, got %T", args[1])
		}
		length, ok := toFloat64(args[2])
		if !ok {
			return nil, fmt.Errorf("length arg must be a number, got %T", args[2])
		}
		startIdx := int(start) - 1 // convert to 0-based
		return runeSlice(s, startIdx, int(length)), nil
	}
	registerFunction("MID", func(args ...any) (any, error) {
		v, err := mid(args...)
		if err != nil {
			return nil, fmt.Errorf("MID %v", err)
		}
		return v, nil
	})
	registerFunction("SUBSTR", func(args ...any) (any, error) {
		v, err := mid(args...)
		if err != nil {
			return nil, fmt.Errorf("SUBSTR %v", err)
		}
		return v, nil
	})

	registerFunction("REPLACE", func(args ...any) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("REPLACE requires 3 arguments")
		}
		s := toString(args[0])
		old := toString(args[1])
		newStr := toString(args[2])
		return strings.ReplaceAll(s, old, newStr), nil
	})

	// FIND(needle, haystack) returns 1-based position; 0 if not found.
	registerFunction("FIND", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("FIND requires 2 arguments")
		}
		needle := toString(args[0])
		haystack := toString(args[1])
		idx := strings.Index(haystack, needle)
		if idx < 0 {
			return 0.0, nil
		}
		// Convert byte index to 1-based rune position.
		return float64(utf8.RuneCountInString(haystack[:idx]) + 1), nil
	})

	registerFunction("CONTAINS", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("CONTAINS requires 2 arguments")
		}
		return strings.Contains(toString(args[0]), toString(args[1])), nil
	})

	registerFunction("STARTSWITH", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("STARTSWITH requires 2 arguments")
		}
		return strings.HasPrefix(toString(args[0]), toString(args[1])), nil
	})

	registerFunction("ENDSWITH", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("ENDSWITH requires 2 arguments")
		}
		return strings.HasSuffix(toString(args[0]), toString(args[1])), nil
	})

	registerFunction("REGEX_MATCH", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("REGEX_MATCH requires 2 arguments (string, pattern)")
		}
		s := toString(args[0])
		pat := toString(args[1])
		re, err := regexp.Compile(pat)
		if err != nil {
			return nil, fmt.Errorf("REGEX_MATCH: invalid pattern: %w", err)
		}
		return re.MatchString(s), nil
	})

	registerFunction("REPEAT", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("REPEAT requires 2 arguments")
		}
		s := toString(args[0])
		n, ok := toFloat64(args[1])
		if !ok {
			return nil, fmt.Errorf("REPEAT: count arg must be a number, got %T", args[1])
		}
		count := int(n)
		if count < 0 {
			return nil, fmt.Errorf("REPEAT: negative count %d", count)
		}
		return strings.Repeat(s, count), nil
	})
}
