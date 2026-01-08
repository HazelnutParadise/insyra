package insyra

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/internal/core"
)

type MergeMode int

const (
	MergeModeInner MergeMode = iota
	MergeModeOuter
	MergeModeLeft
	MergeModeRight
)

type MergeDirection int

const (
	MergeDirectionHorizontal MergeDirection = iota
	MergeDirectionVertical
)

// Merge merges two DataTables based on a key column or row name.
// direction: MergeDirectionHorizontal (join columns) or MergeDirectionVertical (join rows)
// mode: MergeModeInner, MergeModeOuter, MergeModeLeft, or MergeModeRight
// on: (Optional) the name of the column to join on (for horizontal). If empty or omitted, uses row names.
func (dt *DataTable) Merge(other IDataTable, direction MergeDirection, mode MergeMode, on ...string) (*DataTable, error) {
	var onLeft, onRight string
	if len(on) > 2 {
		return nil, fmt.Errorf("at most two 'on' parameters are allowed")
	}
	if len(on) == 1 {
		onLeft = on[0]
		onRight = on[0]
	}
	if len(on) == 2 {
		onLeft = on[0]
		onRight = on[1]
	}
	switch direction {
	case MergeDirectionHorizontal:
		return dt.mergeHorizontal(other, onLeft, onRight, mode, "column")
	case MergeDirectionVertical:
		return dt.mergeVertical(other, mode)
	}
	return nil, fmt.Errorf("invalid direction: %v", direction)
}

func (dt *DataTable) mergeHorizontal(other IDataTable, onLeft, onRight string, mode MergeMode, keyType string) (*DataTable, error) {
	var result *DataTable
	var err error

	dt.AtomicDo(func(d *DataTable) {
		other.AtomicDo(func(o *DataTable) {
			colIdx1 := -1
			if onLeft == "" && onRight == "" {
				// Use row names, no column index needed
			} else {
				for i, col := range d.columns {
					if col.name == onLeft {
						colIdx1 = i
						break
					}
				}
				if colIdx1 == -1 {
					err = fmt.Errorf("%s %s not found in first table", keyType, onLeft)
					return
				}
			}

			colIdx2 := -1
			if onLeft == "" && onRight == "" {
				// Use row names
			} else {
				for i, col := range o.columns {
					if col.name == onRight {
						colIdx2 = i
						break
					}
				}
				if colIdx2 == -1 {
					err = fmt.Errorf("%s %s not found in second table", keyType, onRight)
					return
				}
			}

			// Map values to row indices
			// Use string representation as key for safety with any type
			map1 := make(map[string][]int)
			valMap1 := make(map[string]any)
			var nameless1 []int
			if onLeft == "" && onRight == "" {
				// Use row names
				for i := 0; i < d.getMaxColLength(); i++ {
					if name, ok := d.getRowNameByIndex(i); ok {
						map1[name] = append(map1[name], i)
						valMap1[name] = name
					} else {
						nameless1 = append(nameless1, i)
					}
				}
			} else {
				for i, val := range d.columns[colIdx1].data {
					s := fmt.Sprintf("%v", val)
					map1[s] = append(map1[s], i)
					valMap1[s] = val
				}
			}

			map2 := make(map[string][]int)
			valMap2 := make(map[string]any)
			var nameless2 []int
			if onLeft == "" && onRight == "" {
				// Use row names
				for i := 0; i < o.getMaxColLength(); i++ {
					if name, ok := o.getRowNameByIndex(i); ok {
						map2[name] = append(map2[name], i)
						valMap2[name] = name
					} else {
						nameless2 = append(nameless2, i)
					}
				}
			} else {
				for i, val := range o.columns[colIdx2].data {
					s := fmt.Sprintf("%v", val)
					map2[s] = append(map2[s], i)
					valMap2[s] = val
				}
			}

			// Determine keys
			var keys []string
			switch mode {
			case MergeModeInner:
				for s := range map1 {
					if _, ok := map2[s]; ok {
						keys = append(keys, s)
					}
				}
			case MergeModeOuter:
				allKeys := make(map[string]bool)
				for s := range map1 {
					allKeys[s] = true
				}
				for s := range map2 {
					allKeys[s] = true
				}
				for s := range allKeys {
					keys = append(keys, s)
				}
			case MergeModeLeft:
				for s := range map1 {
					keys = append(keys, s)
				}
			case MergeModeRight:
				for s := range map2 {
					keys = append(keys, s)
				}
			default:
				err = fmt.Errorf("invalid mode: %v", mode)
				return
			}

			// Construct new columns
			newCols := make([]*DataList, 0)
			for _, col := range d.columns {
				newCols = append(newCols, NewDataList().SetName(col.name))
			}
			for i, col := range o.columns {
				if i == colIdx2 {
					continue
				}
				// Handle name collision
				newName := col.name
				for {
					_, ok := d.getColNumberByName_notAtomic(newName)
					if !ok {
						break
					}
					newName = newName + "_other"
				}
				newCols = append(newCols, NewDataList().SetName(newName))
			}

			// Collect row names
			resultRowNames := core.NewBiIndex(0)
			currentRowIdx := 0

			// Helper to ensure unique row names in the result set
			ensureUniqueName := func(name string) string {
				if name == "" {
					return ""
				}
				originalName := name
				counter := 1
				for resultRowNames.Has(name) {
					name = fmt.Sprintf("%s_%d", originalName, counter)
					counter++
				}
				return name
			}

			// Fill data
			for _, s := range keys {
				indices1 := map1[s]
				indices2 := map2[s]

				if len(indices1) > 0 && len(indices2) > 0 {
					for _, idx1 := range indices1 {
						for _, idx2 := range indices2 {
							// Preserve row name
							var rowName string
							if onLeft != "" || onRight != "" {
								if name, ok := d.getRowNameByIndex(idx1); ok {
									rowName = name
								} else if name, ok := o.getRowNameByIndex(idx2); ok {
									rowName = name
								}
							} else {
								rowName = s
							}

							if rowName != "" {
								uniqueName := ensureUniqueName(rowName)
								_, _ = resultRowNames.Set(currentRowIdx, uniqueName)
							}

							colOffset := 0
							for _, col := range d.columns {
								newCols[colOffset].Append(col.data[idx1])
								colOffset++
							}
							for i, col := range o.columns {
								if i == colIdx2 {
									continue
								}
								newCols[colOffset].Append(col.data[idx2])
								colOffset++
							}
							currentRowIdx++
						}
					}
				} else if len(indices1) > 0 && (mode == MergeModeOuter || mode == MergeModeLeft) {
					for _, idx1 := range indices1 {
						var rowName string
						if onLeft != "" || onRight != "" {
							if name, ok := d.getRowNameByIndex(idx1); ok {
								rowName = name
							}
						} else {
							rowName = s
						}

						if rowName != "" {
							uniqueName := ensureUniqueName(rowName)
							_, _ = resultRowNames.Set(currentRowIdx, uniqueName)
						}

						colOffset := 0
						for _, col := range d.columns {
							newCols[colOffset].Append(col.data[idx1])
							colOffset++
						}
						for i := range o.columns {
							if i == colIdx2 {
								continue
							}
							newCols[colOffset].Append(nil)
							colOffset++
						}
						currentRowIdx++
					}
				} else if len(indices2) > 0 && (mode == MergeModeOuter || mode == MergeModeRight) {
					for _, idx2 := range indices2 {
						var rowName string
						if onLeft != "" || onRight != "" {
							if name, ok := o.getRowNameByIndex(idx2); ok {
								rowName = name
							}
						} else {
							rowName = s
						}

						if rowName != "" {
							uniqueName := ensureUniqueName(rowName)
							_, _ = resultRowNames.Set(currentRowIdx, uniqueName)
						}

						colOffset := 0
						for range d.columns {
							if colOffset == colIdx1 {
								newCols[colOffset].Append(valMap2[s])
							} else {
								newCols[colOffset].Append(nil)
							}
							colOffset++
						}
						for i, col := range o.columns {
							if i == colIdx2 {
								continue
							}
							newCols[colOffset].Append(col.data[idx2])
							colOffset++
						}
						currentRowIdx++
					}
				}
			}

			// Add nameless rows (joining by row names)
			if onLeft == "" && onRight == "" {
				if mode == MergeModeOuter || mode == MergeModeLeft {
					for _, idx1 := range nameless1 {
						colOffset := 0
						for _, col := range d.columns {
							newCols[colOffset].Append(col.data[idx1])
							colOffset++
						}
						for i := range o.columns {
							if i == colIdx2 {
								continue
							}
							newCols[colOffset].Append(nil)
							colOffset++
						}
						currentRowIdx++
					}
				}
				if mode == MergeModeOuter || mode == MergeModeRight {
					for _, idx2 := range nameless2 {
						colOffset := 0
						for range d.columns {
							newCols[colOffset].Append(nil)
							colOffset++
						}
						for i, col := range o.columns {
							if i == colIdx2 {
								continue
							}
							newCols[colOffset].Append(col.data[idx2])
							colOffset++
						}
						currentRowIdx++
					}
				}
			}

			result = NewDataTable(newCols...)
			result.rowNames = resultRowNames
		})
	})

	return result, err
}

func (dt *DataTable) mergeVertical(other IDataTable, mode MergeMode) (*DataTable, error) {
	otherDT, ok := other.(*DataTable)
	if !ok {
		return nil, fmt.Errorf("other must be a *DataTable")
	}

	var result *DataTable
	dt.AtomicDo(func(d *DataTable) {
		otherDT.AtomicDo(func(o *DataTable) {
			colNames1 := d.ColNames()
			colNames2 := o.ColNames()

			var finalColNames []string
			if mode == MergeModeInner {
				set2 := make(map[string]bool)
				for _, name := range colNames2 {
					set2[name] = true
				}
				for _, name := range colNames1 {
					if set2[name] {
						finalColNames = append(finalColNames, name)
					}
				}
			} else {
				set := make(map[string]bool)
				for _, name := range colNames1 {
					if !set[name] {
						finalColNames = append(finalColNames, name)
						set[name] = true
					}
				}
				for _, name := range colNames2 {
					if !set[name] {
						finalColNames = append(finalColNames, name)
						set[name] = true
					}
				}
			}

			if len(finalColNames) == 0 {
				result = NewDataTable()
				return
			}

			newCols := make([]*DataList, len(finalColNames))
			for i, name := range finalColNames {
				newCols[i] = NewDataList().SetName(name)
			}
			result = NewDataTable(newCols...)

			// Helper to get column index by name (not atomic)
			getColIdx := func(dt *DataTable, name string) int {
				for i, col := range dt.columns {
					if col.name == name {
						return i
					}
				}
				return -1
			}

			// Append rows from d
			dMax := d.getMaxColLength()
			for i := 0; i < dMax; i++ {
				for j, name := range finalColNames {
					idx := getColIdx(d, name)
					var val any = nil
					if idx != -1 && i < len(d.columns[idx].data) {
						val = d.columns[idx].data[i]
					}
					result.columns[j].data = append(result.columns[j].data, val)
				}
				if name, ok := d.getRowNameByIndex(i); ok {
					if unique := safeRowName(result, name); unique != "" {
						_, _ = result.rowNames.Set(i, unique)
					}
				}
			}

			// Append rows from o
			oMax := o.getMaxColLength()
			currentRows := result.getMaxColLength()
			for i := 0; i < oMax; i++ {
				for j, name := range finalColNames {
					idx := getColIdx(o, name)
					var val any = nil
					if idx != -1 && i < len(o.columns[idx].data) {
						val = o.columns[idx].data[i]
					}
					result.columns[j].data = append(result.columns[j].data, val)
				}
				if name, ok := o.getRowNameByIndex(i); ok {
					if unique := safeRowName(result, name); unique != "" {
						_, _ = result.rowNames.Set(currentRows+i, unique)
					}
				}
			}
		})
	})
	return result, nil
}
