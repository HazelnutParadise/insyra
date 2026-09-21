package insyra

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/internal/algorithms"
)

// DataTableSortConfig selects one sort level: the column to sort by and its
// direction. Col takes the library's column selector, so it is an Excel-style
// index string ("A", "B", ... "AA"), a Name, or an int position. A config that
// leaves Col nil sorts by the first column.
type DataTableSortConfig struct {
	Col        any  // The column to sort by. nil sorts by the first column.
	Descending bool // Whether to sort in descending order, default is ascending
}

// SortBy sorts the DataTable based on multiple columns as specified in the configs.
// Supports sorting by column index (A, B, ...), column number (0, 1, ...), or column name.
// For multi-column sorting, the order of configs determines the priority (first config has highest priority).
//
// Every level is resolved before any row moves. A level that names a column
// that is not there records the error on SortBy and leaves the table
// unchanged, rather than sorting by the levels that happened to be valid.
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
	// A config that picks no column sorts by the first one, which is what the
	// zero value has always meant.
	if config.Col == nil {
		if len(dt.columns) == 0 {
			return -1, formatSortProblem(level, "the table has no columns")
		}
		return 0, ""
	}
	num, warning, problem := dt.lookupColSelector(config.Col)
	if problem != "" {
		return -1, formatSortProblem(level, "%s", problem)
	}
	if warning != "" {
		dt.warn("SortBy", "level %d: %s", level, warning)
	}
	return num, ""
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
