package insyra

import (
	"fmt"
	"math"
)

// Replace all occurrences of oldValue with newValue in the DataTable.
func (dt *DataTable) Replace(oldValue, newValue any) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		dt.replace_notAtomic(oldValue, newValue)
	})
	return dt
}

// Replace all occurrences of NaN with newValue in the DataTable.
func (dt *DataTable) ReplaceNaNsWith(newValue any) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		dt.replaceNaNsWith_notAtomic(newValue)
	})
	return dt
}

// Replace all occurrences of nil with newValue in the DataTable.
func (dt *DataTable) ReplaceNilsWith(newValue any) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		dt.replace_notAtomic(nil, newValue)
	})
	return dt
}

// Replace all occurrences of NaN and nil with newValue in the DataTable.
func (dt *DataTable) ReplaceNaNsAndNilsWith(newValue any) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		dt.replaceNaNsAndNilsWith_notAtomic(newValue)
	})
	return dt
}

// Replace occurrences of oldValue with newValue in a specific row of the DataTable.
//
// # Parameters
//   - rowIndex: The index of the row to perform replacements in.
//   - oldValue: The value to be replaced.
//   - newValue: The value to replace with.
//   - mode (optional): An integer indicating the replacement mode.
//   - 0 (default): Replace all occurrences.
//   - 1: Replace only the first occurrence.
//   - -1: Replace only the last occurrence.
func (dt *DataTable) ReplaceInRow(rowIndex int, oldValue, newValue any, mode ...int) *DataTable {
	var err error
	dt.AtomicDo(func(dt *DataTable) {
		err = dt.replaceInRow_notAtomic(rowIndex, oldValue, newValue, mode...)
	})
	if err != nil {
		dt.fail("ReplaceInRow", "Error: %s", err.Error())
	}
	return dt
}

// Replace occurrences of NaN with newValue in a specific row of the DataTable.
//
// # Parameters
//   - rowIndex: The index of the row to perform replacements in.
//   - newValue: The value to replace NaNs with.
//   - mode (optional): An integer indicating the replacement mode.
//   - 0 (default): Replace all occurrences.
//   - 1: Replace only the first occurrence.
//   - -1: Replace only the last occurrence.
func (dt *DataTable) ReplaceNaNsInRow(rowIndex int, newValue any, mode ...int) *DataTable {
	var err error
	dt.AtomicDo(func(dt *DataTable) {
		err = dt.replaceInRow_notAtomic(rowIndex, math.NaN(), newValue, mode...)
	})
	if err != nil {
		dt.fail("ReplaceNaNsInRow", "Error: %s", err.Error())
	}
	return dt
}

// Replace occurrences of nil with newValue in a specific row of the DataTable.
//
// # Parameters
//   - rowIndex: The index of the row to perform replacements in.
//   - newValue: The value to replace nils with.
//   - mode (optional): An integer indicating the replacement mode.
//   - 0 (default): Replace all occurrences.
//   - 1: Replace only the first occurrence.
//   - -1: Replace only the last occurrence.
func (dt *DataTable) ReplaceNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable {
	var err error
	dt.AtomicDo(func(dt *DataTable) {
		err = dt.replaceInRow_notAtomic(rowIndex, nil, newValue, mode...)
	})
	if err != nil {
		dt.fail("ReplaceNilsInRow", "Error: %s", err.Error())
	}
	return dt
}

// Replace occurrences of NaN and nil with newValue in a specific row of the DataTable.
//
// # Parameters
//   - rowIndex: The index of the row to perform replacements in.
//   - newValue: The value to replace NaNs and nils with.
//   - mode (optional): An integer indicating the replacement mode.
//   - 0 (default): Replace all occurrences.
//   - 1: Replace only the first occurrence.
//   - -1: Replace only the last occurrence.
func (dt *DataTable) ReplaceNaNsAndNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable {
	var err error
	dt.AtomicDo(func(dt *DataTable) {
		err = dt.replaceNaNsAndNilsInRow_notAtomic(rowIndex, newValue, mode...)
	})
	if err != nil {
		dt.fail("ReplaceNaNsAndNilsInRow", "Error: %s", err.Error())
	}
	return dt
}

// Replace occurrences of oldValue with newValue in a specific column of the DataTable.
//
// # Parameters
//   - colIndex: The index or name of the column to perform replacements in.
//   - oldValue: The value to be replaced.
//   - newValue: The value to replace with.
//   - mode (optional): An integer indicating the replacement mode.
//   - 0 (default): Replace all occurrences.
//   - 1: Replace only the first occurrence.
//   - -1: Replace only the last occurrence.
func (dt *DataTable) ReplaceInCol(col any, oldValue, newValue any, mode ...int) *DataTable {
	return dt.replaceInCol("ReplaceInCol", col, oldValue, newValue, mode...)
}

// ReplaceInColByName is ReplaceInCol for a column addressed by name.
func (dt *DataTable) ReplaceInColByName(name string, oldValue, newValue any, mode ...int) *DataTable {
	return dt.replaceInCol("ReplaceInColByName", Name(name), oldValue, newValue, mode...)
}

func (dt *DataTable) replaceInCol(funcName string, col any, oldValue, newValue any, mode ...int) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		num, ok := dt.resolveColSelector(funcName, col)
		if !ok {
			return
		}
		if err := dt.replaceInCol_notAtomic(num, oldValue, newValue, mode...); err != nil {
			dt.fail(funcName, "%s", err.Error())
		}
	})
	return dt
}

// Replace occurrences of NaN with newValue in a specific column of the DataTable.
//
// # Parameters
//   - colIndex: The index or name of the column to perform replacements in.
//   - newValue: The value to replace NaNs with.
//   - mode (optional): An integer indicating the replacement mode.
//   - 0 (default): Replace all occurrences.
//   - 1: Replace only the first occurrence.
//   - -1: Replace only the last occurrence.
func (dt *DataTable) ReplaceNaNsInCol(col any, newValue any, mode ...int) *DataTable {
	return dt.replaceNaNsInCol("ReplaceNaNsInCol", col, newValue, mode...)
}

// ReplaceNaNsInColByName is ReplaceNaNsInCol for a column addressed by name.
func (dt *DataTable) ReplaceNaNsInColByName(name string, newValue any, mode ...int) *DataTable {
	return dt.replaceNaNsInCol("ReplaceNaNsInColByName", Name(name), newValue, mode...)
}

func (dt *DataTable) replaceNaNsInCol(funcName string, col any, newValue any, mode ...int) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		num, ok := dt.resolveColSelector(funcName, col)
		if !ok {
			return
		}
		if err := dt.replaceInCol_notAtomic(num, math.NaN(), newValue, mode...); err != nil {
			dt.fail(funcName, "%s", err.Error())
		}
	})
	return dt
}

// ReplaceNilsInCol replaces occurrences of nil with newValue in a specific column of the DataTable.
//
// # Parameters
//   - colIndex: The index of the column to perform replacements in.
//   - newValue: The value to replace nils with.
//   - mode (optional): An integer indicating the replacement mode.
//   - 0 (default): Replace all occurrences.
//   - 1: Replace only the first occurrence.
//   - -1: Replace only the last occurrence.
func (dt *DataTable) ReplaceNilsInCol(col any, newValue any, mode ...int) *DataTable {
	return dt.replaceNilsInCol("ReplaceNilsInCol", col, newValue, mode...)
}

// ReplaceNilsInColByName is ReplaceNilsInCol for a column addressed by name.
func (dt *DataTable) ReplaceNilsInColByName(name string, newValue any, mode ...int) *DataTable {
	return dt.replaceNilsInCol("ReplaceNilsInColByName", Name(name), newValue, mode...)
}

func (dt *DataTable) replaceNilsInCol(funcName string, col any, newValue any, mode ...int) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		num, ok := dt.resolveColSelector(funcName, col)
		if !ok {
			return
		}
		if err := dt.replaceInCol_notAtomic(num, nil, newValue, mode...); err != nil {
			dt.fail(funcName, "%s", err.Error())
		}
	})
	return dt
}

// ReplaceNaNsAndNilsInCol replaces occurrences of NaN and nil with newValue in a specific column of the DataTable.
//
// # Parameters
//   - colIndex: The index of the column to perform replacements in.
//   - newValue: The value to replace NaNs and nils with.
//   - mode (optional): An integer indicating the replacement mode.
//   - 0 (default): Replace all occurrences.
//   - 1: Replace only the first occurrence.
//   - -1: Replace only the last occurrence.
func (dt *DataTable) ReplaceNaNsAndNilsInCol(col any, newValue any, mode ...int) *DataTable {
	return dt.replaceNaNsAndNilsInCol("ReplaceNaNsAndNilsInCol", col, newValue, mode...)
}

// ReplaceNaNsAndNilsInColByName is ReplaceNaNsAndNilsInCol for a column addressed by name.
func (dt *DataTable) ReplaceNaNsAndNilsInColByName(name string, newValue any, mode ...int) *DataTable {
	return dt.replaceNaNsAndNilsInCol("ReplaceNaNsAndNilsInColByName", Name(name), newValue, mode...)
}

func (dt *DataTable) replaceNaNsAndNilsInCol(funcName string, col any, newValue any, mode ...int) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		num, ok := dt.resolveColSelector(funcName, col)
		if !ok {
			return
		}
		if err := dt.replaceNaNsAndNilsInCol_notAtomic(num, newValue, mode...); err != nil {
			dt.fail(funcName, "%s", err.Error())
		}
	})
	return dt
}

// ========================== Not Atomic Versions ==========================

func (dt *DataTable) replace_notAtomic(oldValue, newValue any) {
	defer dt.updateTimestamp()
	for _, col := range dt.columns {
		col.replaceAll_notAtomic(oldValue, newValue)
	}
}

func (dt *DataTable) replaceNaNsWith_notAtomic(newValue any) {
	defer dt.updateTimestamp()
	for _, col := range dt.columns {
		col.replaceAll_notAtomic(math.NaN(), newValue)
	}
}

func (dt *DataTable) replaceNaNsAndNilsWith_notAtomic(newValue any) {
	defer dt.updateTimestamp()
	for i, col := range dt.columns {
		for j, cell := range col.data {
			// Route through isNilOrNaN so float32 NaN is handled too (a bare
			// float64 assertion missed it), consistent with impute/pivot.
			if isNilOrNaN(cell) {
				dt.columns[i].data[j] = newValue
			}
		}
	}
}

func (dt *DataTable) replaceInRow_notAtomic(rowIndex int, oldValue, newValue any, mode ...int) error {
	defer dt.updateTimestamp()
	modeFlag := 0
	if len(mode) > 1 {
		return fmt.Errorf("mode parameter can only have 0 or 1 value")
	}
	if len(mode) == 1 {
		modeFlag = mode[0]
	}

	matches := valueMatcher(oldValue)
	switch modeFlag {
	case 1:
		// 取代第一個
		for _, col := range dt.columns {
			if rowIndex >= 0 && rowIndex < len(col.data) && matches(col.data[rowIndex]) {
				col.data[rowIndex] = newValue
				col.updateTimestamp()
				break
			}
		}
	case 0: // 取代所有符合的
		for _, col := range dt.columns {
			if rowIndex >= 0 && rowIndex < len(col.data) && matches(col.data[rowIndex]) {
				col.data[rowIndex] = newValue
				col.updateTimestamp()
			}
		}
	case -1: // 從後往前取代第一個
		for i := len(dt.columns) - 1; i >= 0; i-- {
			col := dt.columns[i]
			if rowIndex >= 0 && rowIndex < len(col.data) && matches(col.data[rowIndex]) {
				col.data[rowIndex] = newValue
				col.updateTimestamp()
				break
			}
		}
	default:
		return fmt.Errorf("invalid mode parameter, no replacements made")
	}
	return nil
}

func (dt *DataTable) replaceNaNsAndNilsInRow_notAtomic(rowIndex int, newValue any, mode ...int) error {
	defer dt.updateTimestamp()
	modeFlag := 0
	if len(mode) > 1 {
		return fmt.Errorf("mode parameter can only have 0 or 1 value")
	}
	if len(mode) == 1 {
		modeFlag = mode[0]
	}
	switch modeFlag {
	case 1:
		// 取代第一個
		for _, col := range dt.columns {
			if rowIndex >= 0 && rowIndex < len(col.data) {
				if col.data[rowIndex] == nil {
					col.data[rowIndex] = newValue
					col.updateTimestamp()
					break
				} else if val, ok := col.data[rowIndex].(float64); ok {
					if math.IsNaN(val) {
						col.data[rowIndex] = newValue
						col.updateTimestamp()
						break
					}
				}
			}
		}
	case 0: // 取代所有符合的
		for _, col := range dt.columns {
			if rowIndex >= 0 && rowIndex < len(col.data) {
				if col.data[rowIndex] == nil {
					col.data[rowIndex] = newValue
					col.updateTimestamp()
				} else if val, ok := col.data[rowIndex].(float64); ok {
					if math.IsNaN(val) {
						col.data[rowIndex] = newValue
						col.updateTimestamp()
					}
				}
			}
		}
	case -1: // 從後往前取代第一個
		for i := len(dt.columns) - 1; i >= 0; i-- {
			col := dt.columns[i]
			if rowIndex >= 0 && rowIndex < len(col.data) {
				if col.data[rowIndex] == nil {
					col.data[rowIndex] = newValue
					col.updateTimestamp()
					break
				} else if val, ok := col.data[rowIndex].(float64); ok {
					if math.IsNaN(val) {
						col.data[rowIndex] = newValue
						col.updateTimestamp()
						break
					}
				}
			}
		}
	default:
		return fmt.Errorf("invalid mode parameter, no replacements made")
	}
	return nil
}

func (dt *DataTable) replaceInCol_notAtomic(colNo int, oldValue, newValue any, mode ...int) error {
	modeFlag := 0
	if len(mode) > 1 {
		return fmt.Errorf("mode parameter can only have 0 or 1 value")
	}
	if len(mode) == 1 {
		modeFlag = mode[0]
	}
	switch modeFlag {
	case 1:
		// 取代第一個
		dt.columns[colNo].replaceFirst_notAtomic(oldValue, newValue)
	case 0: // 取代所有符合的
		dt.columns[colNo].replaceAll_notAtomic(oldValue, newValue)
	case -1: // 從後往前取代第一個
		dt.columns[colNo].replaceLast_notAtomic(oldValue, newValue)
	default:
		return fmt.Errorf("invalid mode parameter, no replacements made")
	}
	return nil
}

func (dt *DataTable) replaceNaNsAndNilsInCol_notAtomic(colNo int, newValue any, mode ...int) error {
	modeFlag := 0
	if len(mode) > 1 {
		return fmt.Errorf("mode parameter can only have 0 or 1 value")
	}
	if len(mode) == 1 {
		modeFlag = mode[0]
	}
	switch modeFlag {
	case 1:
		// 取代第一個
		for i, val := range dt.columns[colNo].data {
			if val == nil {
				dt.columns[colNo].data[i] = newValue
				dt.columns[colNo].updateTimestamp()
				break
			} else if v, ok := val.(float64); ok && math.IsNaN(v) {
				dt.columns[colNo].data[i] = newValue
				dt.columns[colNo].updateTimestamp()
				break
			}
		}
	case 0: // 取代所有符合的
		dt.columns[colNo].replaceNaNsAndNilsWith_notAtomic(newValue)
	case -1: // 從後往前取代第一個
		for i := len(dt.columns[colNo].data) - 1; i >= 0; i-- {
			val := dt.columns[colNo].data[i]
			if val == nil {
				dt.columns[colNo].data[i] = newValue
				dt.columns[colNo].updateTimestamp()
				break
			} else if v, ok := val.(float64); ok && math.IsNaN(v) {
				dt.columns[colNo].data[i] = newValue
				dt.columns[colNo].updateTimestamp()
				break
			}
		}
	default:
		return fmt.Errorf("invalid mode parameter, no replacements made")
	}
	return nil
}
