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
