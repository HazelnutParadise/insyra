package insyra

import (
	"fmt"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/algorithms"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

// DataTableSortConfig selects one sort level: the column to sort by and its
// direction. Give one of ColumnIndex, ColumnName and ColumnNumber. When more
// than one is given, the first in that order is used and SortBy logs a
// warning naming the others — the same index-before-name rule mkt's configs
// follow.
type DataTableSortConfig struct {
	ColumnIndex string // The column index (A, B, C, ...). Takes precedence over ColumnName and ColumnNumber.
	// ColumnNumber is the column's 0-based position, used only when
	// ColumnIndex and ColumnName are empty. Zero is the same value as not
	// setting it, so a config that sets only ColumnNumber: 0 names no column
	// and is refused; select the first column with ColumnIndex: "A".
	ColumnNumber int
	ColumnName   string // The column name. Takes precedence over ColumnNumber.
	Descending   bool   // Whether to sort in descending order, default is ascending
}

// SortBy sorts the DataTable based on multiple columns as specified in the configs.
// Supports sorting by column index (A, B, ...), column number (0, 1, ...), or column name.
// For multi-column sorting, the order of configs determines the priority (first config has highest priority).
//
// Every level is resolved before any row moves. A level that names no column,
// or a column that is not there, records the error on SortBy and leaves the
// table unchanged, rather than sorting by the levels that happened to be valid.
func (dt *DataTable) SortBy(configs ...DataTableSortConfig) *DataTable {
	if len(configs) == 0 {
		dt.warn("SortBy", "No sorting configuration provided, returning original DataTable.")
		return dt
	}
	dt.AtomicDo(func(t *DataTable) {
		positions := make([]int, len(configs))
		for i, config := range configs {
			pos, problem := t.sortColumnPosition(i+1, config)
			if problem != "" {
				t.fail("SortBy", "%s", problem)
				return
			}
			positions[i] = pos
		}

		// Stable sort from the last level to the first. Each level reads its
		// column when it runs, because an earlier pass has already moved rows.
		for i := len(configs) - 1; i >= 0; i-- {
			t.sortRowsByColumn(positions[i], configs[i].Descending)
		}
	})
	return dt
}

// sortColumnPosition resolves one sort level to a column position, or
// describes why it cannot. It records nothing: SortBy reports the problem as
// its own, so the error names the call the user made rather than an internal
// lookup.
func (dt *DataTable) sortColumnPosition(level int, config DataTableSortConfig) (int, string) {
	var given []string
	if config.ColumnIndex != "" {
		given = append(given, "ColumnIndex")
	}
	if config.ColumnName != "" {
		given = append(given, "ColumnName")
	}
	if config.ColumnNumber != 0 {
		given = append(given, "ColumnNumber")
	}

	if len(given) == 0 {
		return -1, formatSortProblem(level, `names no column; ColumnNumber: 0 is the same value as not setting it, so select the first column with ColumnIndex: "A"`)
	}
	if len(given) > 1 {
		dt.warn("SortBy", "level %d gives %s; sorting by %s and ignoring %s",
			level, strings.Join(given, ", "), given[0], strings.Join(given[1:], ", "))
	}

	switch given[0] {
	case "ColumnIndex":
		upper := strings.ToUpper(config.ColumnIndex)
		if pos, ok := utils.ParseColIndex(upper); ok && pos >= 0 && pos < len(dt.columns) {
			return pos, ""
		}
		// The same name fallback colSilently and GetCol use. The owner has
		// ruled it should go (#225); until then SortBy keeps it, so the three
		// agree.
		for pos, column := range dt.columns {
			if column.name == upper {
				return pos, ""
			}
		}
		return -1, formatSortProblem(level, "ColumnIndex %q matches no column", config.ColumnIndex)
	case "ColumnName":
		for pos, column := range dt.columns {
			if column.name == config.ColumnName {
				return pos, ""
			}
		}
		return -1, formatSortProblem(level, "no column is named %q", config.ColumnName)
	default:
		if config.ColumnNumber < 0 || config.ColumnNumber >= len(dt.columns) {
			return -1, formatSortProblem(level, "ColumnNumber %d is out of range (the table has %d columns)", config.ColumnNumber, len(dt.columns))
		}
		return config.ColumnNumber, ""
	}
}

func formatSortProblem(level int, msg string, args ...any) string {
	return fmt.Sprintf("sort level %d ", level) + fmt.Sprintf(msg, args...)
}

// sortRowsByColumn stably reorders every row by the column at pos.
func (dt *DataTable) sortRowsByColumn(pos int, descending bool) {
	// Permute ALL rows (max column length), not just the sort column's
	// length, so rows are not dropped on a jagged table; index the sort
	// column defensively (treat out-of-range as nil) instead of panicking.
	sortData := dt.columns[pos].data
	n := dt.getMaxColLength()
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	at := func(i int) any {
		if i >= 0 && i < len(sortData) {
			return sortData[i]
		}
		return nil
	}
	algorithms.ParallelSortStableFunc(indices, func(a, b int) int {
		cmp := algorithms.CompareAny(at(a), at(b))
		if descending {
			return -cmp
		}
		return cmp
	})
	// Calculate swaps to apply the permutation using SwapRowsByIndex
	swaps := [][2]int{}
	visited := make([]bool, n)
	for i := range n {
		if !visited[i] && indices[i] != i {
			cycle := []int{}
			j := i
			for !visited[j] {
				visited[j] = true
				cycle = append(cycle, j)
				j = indices[j]
			}
			// Decompose cycle into swaps
			for k := 0; k < len(cycle)-1; k++ {
				swaps = append(swaps, [2]int{cycle[k], cycle[k+1]})
			}
		}
	}
	// Apply swaps to move whole rows
	for _, swap := range swaps {
		dt.SwapRowsByIndex(swap[0], swap[1])
	}
}
