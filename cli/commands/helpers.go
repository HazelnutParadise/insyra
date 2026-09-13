package commands

import (
	"fmt"
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

// parseFloatArg reads a numeric argument and reports a bad one the way the rest
// of the CLI reports things. Returning strconv's own error handed the user
// `strconv.ParseFloat: parsing "abc": invalid syntax`, which names neither the
// command nor the argument.
func parseFloatArg(cmd, field, raw string) (float64, error) {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid %s %q, expected a number", cmd, field, raw)
	}
	return v, nil
}

// parseIntArg is parseFloatArg for a whole number.
func parseIntArg(cmd, field, raw string) (int, error) {
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid %s %q, expected a whole number", cmd, field, raw)
	}
	return v, nil
}
