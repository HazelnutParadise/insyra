package quant

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
)

// resolveColumn resolves a column selector with the library's one rule, for
// every quant function that picks table columns: a
// string is an Excel-style index, insyra.Name is a name compared exactly, an
// int is a 0-based position counting from the end when negative. It returns the
// column's position and its name (its Excel-style index when it has none).
// Call it inside AtomicDo.
func resolveColumn(dt *insyra.DataTable, sel any, role string) (int, string, error) {
	names := dt.ColNames()
	num := -1
	switch v := sel.(type) {
	case nil:
		return -1, "", fmt.Errorf("%s column: no column given; use an Excel-style index such as \"A\", insyra.Name(\"price\"), or an int position", role)
	case string:
		i, ok := insyra.ColIndexToNumber(v)
		if !ok {
			return -1, "", fmt.Errorf("%s column %q is not an Excel-style column index; write insyra.Name(%q) for a column name", role, v, v)
		}
		if i >= len(names) {
			return -1, "", fmt.Errorf("%s column %q does not exist, the table has %d column(s); write insyra.Name(%q) for a column name", role, v, len(names), v)
		}
		num = i
	case insyra.NameSelector:
		for i, n := range names {
			if n == v.Value() {
				num = i
				break
			}
		}
		if num < 0 {
			return -1, "", fmt.Errorf("%s column: no column is named %q", role, v.Value())
		}
	case int:
		num = v
		if num < 0 {
			num += len(names)
		}
		if num < 0 || num >= len(names) {
			return -1, "", fmt.Errorf("%s column number %d is out of range, the table has %d column(s)", role, v, len(names))
		}
	default:
		return -1, "", fmt.Errorf("%s column selector of type %T; use an Excel-style index such as \"A\", insyra.Name(\"price\"), or an int position", role, sel)
	}
	name := names[num]
	if name == "" {
		name, _ = insyra.ColNumberToIndex(num)
	}
	return num, name, nil
}
