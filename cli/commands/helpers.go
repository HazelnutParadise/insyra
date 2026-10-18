package commands

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func parseAlias(args []string) ([]string, string) {
	if len(args) >= 2 && strings.EqualFold(args[len(args)-2], "as") {
		return args[:len(args)-2], args[len(args)-1]
	}
	return args, "$result"
}

// parseFlexBool accepts true|false|yes|no|on|off|1|0 (case-insensitive).
// Returns an error so the caller can produce an option-specific message
// (e.g. "load: invalid value %q for headers").
func parseFlexBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0":
		return false, nil
	}
	return false, fmt.Errorf("expected true|false|yes|no|1|0, got %q", raw)
}

func parseLiteral(raw string) any {
	if strings.EqualFold(raw, "nil") {
		return nil
	}
	if strings.EqualFold(raw, "true") {
		return true
	}
	if strings.EqualFold(raw, "false") {
		return false
	}
	if integer, err := strconv.Atoi(raw); err == nil {
		return integer
	}
	if decimal, err := strconv.ParseFloat(raw, 64); err == nil {
		return decimal
	}
	return raw
}

func getDataTableVar(ctx *ExecContext, name string) (*insyra.DataTable, error) {
	value, exists := ctx.Vars[name]
	if !exists {
		return nil, fmt.Errorf("variable not found: %s", name)
	}
	table, ok := value.(*insyra.DataTable)
	if !ok {
		return nil, fmt.Errorf("variable %s is not a DataTable", name)
	}
	return table, nil
}

func getDataListVar(ctx *ExecContext, name string) (*insyra.DataList, error) {
	value, exists := ctx.Vars[name]
	if !exists {
		return nil, fmt.Errorf("variable not found: %s", name)
	}
	list, ok := value.(*insyra.DataList)
	if !ok {
		return nil, fmt.Errorf("variable %s is not a DataList", name)
	}
	return list, nil
}

func detectFileKind(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return "csv"
	case ".json":
		return "json"
	case ".xlsx", ".xlsm", ".xls":
		return "excel"
	case ".parquet":
		return "parquet"
	default:
		return ""
	}
}

// varTypeError produces the error for a command that accepts either a
// DataTable or a DataList and matched neither. Reporting "variable not found"
// for a variable that is right there but holds something else sends the caller
// looking for a typo that is not there.
func varTypeError(ctx *ExecContext, cmd, name string) error {
	if ctx == nil || ctx.Vars == nil {
		return fmt.Errorf("%s: variable not found: %s", cmd, name)
	}
	v, ok := ctx.Vars[name]
	if !ok {
		return fmt.Errorf("%s: variable not found: %s", cmd, name)
	}
	return fmt.Errorf("%s: %s holds a %T, which is neither a DataTable nor a DataList", cmd, name, v)
}

// resolveNumberToken lets a numeric argument be the name of a number variable.
// A token that already reads as a number is returned as it is; one that does
// not, and that names a variable holding a number, becomes that number's text.
// Anything else comes back unchanged so the caller reports its usual error.
func resolveNumberToken(ctx *ExecContext, raw string) string {
	if ctx == nil || raw == "" {
		return raw
	}
	if _, err := strconv.ParseFloat(raw, 64); err == nil {
		return raw
	}
	value, ok := ctx.Vars[raw]
	if !ok {
		return raw
	}
	if number, isNumber := numberValue(value); isNumber {
		return strconv.FormatFloat(number, 'g', -1, 64)
	}
	return raw
}

// numberValue reads a variable that holds one number.
func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	}
	return 0, false
}

// atoiArg and parseFloatToken read a numeric command argument, which may name a
// number variable. They return strconv's own error for a bad one, so call sites
// that report it themselves keep doing so.
func atoiArg(ctx *ExecContext, raw string) (int, error) {
	token := resolveNumberToken(ctx, raw)
	if v, err := strconv.Atoi(token); err == nil {
		return v, nil
	}
	// A number variable that holds a whole number as a float64, such as
	// 252.0 from a computation, still counts as a whole number.
	if f, err := strconv.ParseFloat(token, 64); err == nil && f == math.Trunc(f) && math.Abs(f) < 1<<53 {
		return int(f), nil
	}
	return strconv.Atoi(token)
}

func parseFloatToken(ctx *ExecContext, raw string) (float64, error) {
	return strconv.ParseFloat(resolveNumberToken(ctx, raw), 64)
}

// parseFloatArg reads a numeric argument and reports a bad one the way the rest
// of the CLI reports things. Returning strconv's own error handed the user
// `strconv.ParseFloat: parsing "abc": invalid syntax`, which names neither the
// command nor the argument. The argument may name a number variable.
func parseFloatArg(ctx *ExecContext, cmd, field, raw string) (float64, error) {
	v, err := parseFloatToken(ctx, raw)
	if err != nil {
		return 0, wrapArgError(err, "%s: invalid %s %q, expected a number", cmd, field, raw)
	}
	return v, nil
}

// parseIntArg is parseFloatArg for a whole number.
func parseIntArg(ctx *ExecContext, cmd, field, raw string) (int, error) {
	v, err := atoiArg(ctx, raw)
	if err != nil {
		return 0, wrapArgError(err, "%s: invalid %s %q, expected a whole number", cmd, field, raw)
	}
	return v, nil
}

// argError is a message naming the command and the argument that keeps the
// error behind it reachable. The message leaves out strconv's own text, which
// only repeats the input, while errors.Is(err, strconv.ErrSyntax) and
// errors.As still see the cause, as they did when these errors used %w.
type argError struct {
	msg string
	err error
}

func (e *argError) Error() string { return e.msg }
func (e *argError) Unwrap() error { return e.err }

func wrapArgError(err error, format string, args ...any) error {
	return &argError{msg: fmt.Sprintf(format, args...), err: err}
}
