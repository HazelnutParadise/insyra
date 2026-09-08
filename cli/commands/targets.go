package commands

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

// A command that cannot do what it was asked must say so. These helpers check
// the target BEFORE calling the library, so the message names what was missing
// instead of reporting success and leaving the user to notice later.
//
// They read the table's own name lists rather than the Get* lookups, which
// record an error on the table when they miss; a check should not leave a mark
// on the caller's data.

// resolveColumn returns the column name for a selector that is a name, or the
// column number for a numeric selector, verifying that it exists.
func resolveColumn(cmd string, table *insyra.DataTable, selector string) (name string, number int, err error) {
	if n, convErr := strconv.Atoi(selector); convErr == nil {
		if n < 0 || n >= table.NumCols() {
			return "", 0, fmt.Errorf("%s: column %d is out of range (the table has %d columns)", cmd, n, table.NumCols())
		}
		return "", n, nil
	}
	if slices.Contains(table.ColNames(), selector) {
		return selector, -1, nil
	}
	return "", 0, fmt.Errorf("%s: column %q not found (available: %s)", cmd, selector, strings.Join(table.ColNames(), ", "))
}

// requireColumnName fails unless the table has a column with that name.
func requireColumnName(cmd string, table *insyra.DataTable, name string) error {
	if slices.Contains(table.ColNames(), name) {
		return nil
	}
	return fmt.Errorf("%s: column %q not found (available: %s)", cmd, name, strings.Join(table.ColNames(), ", "))
}

// requireRowIndex fails unless index addresses a row; negative indices count
// back from the end, as the DataTable methods do.
func requireRowIndex(cmd string, table *insyra.DataTable, index int) error {
	rows := table.NumRows()
	resolved := index
	if resolved < 0 {
		resolved = rows + resolved
	}
	if resolved < 0 || resolved >= rows {
		return fmt.Errorf("%s: row %d is out of range (the table has %d rows)", cmd, index, rows)
	}
	return nil
}

// requireRowName fails unless the table has a row with that name.
func requireRowName(cmd string, table *insyra.DataTable, name string) error {
	if slices.Contains(table.RowNames(), name) {
		return nil
	}
	return fmt.Errorf("%s: row %q not found", cmd, name)
}

// checkTableErr turns an error the library recorded on the table into a
// returned error, so a failed operation cannot be reported as a success.
func checkTableErr(cmd string, table *insyra.DataTable) error {
	if err := table.PopErr(); err != nil {
		return fmt.Errorf("%s: %v", cmd, err)
	}
	return nil
}

// parseSortDirection accepts the documented spellings and nothing else: a
// misspelling must not silently mean ascending.
func parseSortDirection(raw string) (descending bool, err error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "asc", "ascending":
		return false, nil
	case "desc", "descending":
		return true, nil
	default:
		return false, fmt.Errorf("invalid direction %q (use asc or desc)", raw)
	}
}
