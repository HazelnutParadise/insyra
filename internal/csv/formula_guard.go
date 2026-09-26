package csv

import (
	"strconv"
	"strings"
)

// GuardFormula prefixes text a spreadsheet would execute with a single quote,
// which spreadsheets read as "this is text". Only leading =, +, - and @ (and
// the whitespace a spreadsheet skips before them) start a formula, and text
// that is only a number is left alone: a spreadsheet reads "-5" as the number
// it is and runs nothing. The core CSV writer and csvxl share it so the two
// cannot disagree about what is guarded.
func GuardFormula(s string) string {
	trimmed := strings.TrimLeft(s, " \t\r\n")
	if trimmed == "" {
		return s
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		if _, err := strconv.ParseFloat(strings.TrimSpace(trimmed), 64); err == nil {
			return s
		}
		return "'" + s
	}
	return s
}
