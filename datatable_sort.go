package insyra

import (
	"github.com/HazelnutParadise/insyra/internal/algorithms"
)

type DataTableSortConfig struct {
	ColumnIndex  string // The column index (A, B, C, ...) of the column to sort by, highest priority
	ColumnNumber int    // The column number (0, 1, 2, ...) of the column to sort by, lowest priority
	ColumnName   string // The column name of the column to sort by, second priority
	Descending   bool   // Whether to sort in descending order, default is ascending
}

// SortBy sorts the DataTable based on multiple columns as specified in the configs.
// Supports sorting by column index (A, B, ...), column number (0, 1, ...), or column name.
// For multi-column sorting, the order of configs determines the priority (first config has highest priority).
func (dt *DataTable) SortBy(configs ...DataTableSortConfig) *DataTable {
	if len(configs) == 0 {
		dt.warn("SortBy", "No sorting configuration provided, returning original DataTable.")
		return dt
	}
	dt.AtomicDo(func(t *DataTable) {
		// Reverse configs for multi-column sort (stable sort from last to first)
		reversedConfigs := make([]DataTableSortConfig, len(configs))
		for i, config := range configs {
			reversedConfigs[len(configs)-1-i] = config
		}
		for _, config := range reversedConfigs {
			if config.ColumnIndex == "" && config.ColumnName == "" && config.ColumnNumber < 0 {
				dt.warn("SortBy", "Invalid sorting configuration: %+v, skipping this configuration.", config)
				continue
			}
			var column *DataList
			if config.ColumnIndex != "" {
				column = t.GetCol(config.ColumnIndex)
			} else if config.ColumnName != "" {
				column = t.GetColByName(config.ColumnName)
			} else if config.ColumnNumber >= 0 {
				column = t.GetColByNumber(config.ColumnNumber)
			}
			if column == nil {
				LogWarning("DataTable", "SortBy", "Column not found for config: %+v, skipping.", config)
				continue
			}

			n := len(column.Data())
			indices := make([]int, n)
			for i := range indices {
				indices[i] = i
			}
			algorithms.ParallelSortStableFunc(indices, func(a, b int) int {
				cmp := algorithms.CompareAny(column.Data()[a], column.Data()[b])
				if config.Descending {
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
				t.SwapRowsByIndex(swap[0], swap[1])
			}
		}
	})
	return dt
}
