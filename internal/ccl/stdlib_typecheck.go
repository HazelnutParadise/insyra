package ccl

import (
	"fmt"
	"math"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// registerTypeCheckFunctions registers the functions that say what kind of value
// a cell holds. ISERROR is in the evaluator, since it has to see its argument
// fail, and ISNA is with the other Null/NaN checks.
func registerTypeCheckFunctions() {
	for name, check := range map[string]func(any) bool{
		// An empty cell reads as nil or as empty text, depending on the file.
		// A blank in a numeric column reads as NaN, which ISNA reports.
		"ISBLANK": func(v any) bool { return v == nil || v == "" },
		"ISNUM":   isNumberValue,
		"ISTEXT": func(v any) bool {
			_, ok := v.(string)
			return ok
		},
		"ISBOOL": func(v any) bool {
			_, ok := v.(bool)
			return ok
		},
		// Text that reads as a date counts, because a date read from a CSV file
		// is text and YEAR reads it all the same.
		"ISDATE": func(v any) bool {
			switch x := v.(type) {
			case time.Time:
				return true
			case string:
				_, ok := utils.TryParseTime(x)
				return ok
			}
			return false
		},
	} {
		registerFunction(name, func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("%s requires 1 argument", name)
			}
			return check(args[0]), nil
		})
	}
}

// isNumberValue reports whether v is a number, by its type. NaN is not one, as
// COUNT does not count it, and text is not one even when it reads as a number.
func isNumberValue(v any) bool {
	switch x := v.(type) {
	case float64:
		return !math.IsNaN(x)
	case float32:
		return !math.IsNaN(float64(x))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	}
	return false
}
