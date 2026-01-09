package insyra

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra/internal/utils"
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
		dt.warn("ReplaceInRow", "Error: %s", err.Error())
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
		dt.warn("ReplaceNaNsInRow", "Error: %s", err.Error())
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
		dt.warn("ReplaceNilsInRow", "Error: %s", err.Error())
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
		dt.warn("ReplaceNaNsAndNilsInRow", "Error: %s", err.Error())
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
func (dt *DataTable) ReplaceInCol(colIndex string, oldValue, newValue any, mode ...int) *DataTable {
	var err error
	dt.AtomicDo(func(dt *DataTable) {
		err = dt.replaceInCol_notAtomic(colIndex, oldValue, newValue, mode...)
	})
	if err != nil {
		dt.warn("ReplaceInCol", "Error: %s", err.Error())
	}
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
func (dt *DataTable) ReplaceNaNsInCol(colIndex string, newValue any, mode ...int) *DataTable {
	var err error
	dt.AtomicDo(func(dt *DataTable) {
		err = dt.replaceInCol_notAtomic(colIndex, math.NaN(), newValue, mode...)
	})
	if err != nil {
		dt.warn("ReplaceNaNsInCol", "Error: %s", err.Error())
	}
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
func (dt *DataTable) ReplaceNilsInCol(colIndex string, newValue any, mode ...int) *DataTable {
	var err error
	dt.AtomicDo(func(dt *DataTable) {
		err = dt.replaceInCol_notAtomic(colIndex, nil, newValue, mode...)
	})
	if err != nil {
		dt.warn("ReplaceNilsInCol", "Error: %s", err.Error())
	}
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
func (dt *DataTable) ReplaceNaNsAndNilsInCol(colIndex string, newValue any, mode ...int) *DataTable {
	var err error
	dt.AtomicDo(func(dt *DataTable) {
		err = dt.replaceNaNsAndNilsInCol_notAtomic(colIndex, newValue, mode...)
	})
	if err != nil {
		dt.warn("ReplaceNaNsAndNilsInCol", "Error: %s", err.Error())
	}
	return dt
}

// ========================== Not Atomic Versions ==========================

func (dt *DataTable) replace_notAtomic(oldValue, newValue any) {
	defer func() {
		go dt.updateTimestamp()
	}()
	for _, col := range dt.columns {
		col.replaceAll_notAtomic(oldValue, newValue)
	}
}

func (dt *DataTable) replaceNaNsWith_notAtomic(newValue any) {
	defer func() {
		go dt.updateTimestamp()
	}()
	for _, col := range dt.columns {
		col.replaceAll_notAtomic(math.NaN(), newValue)
	}
}

func (dt *DataTable) replaceNaNsAndNilsWith_notAtomic(newValue any) {
	defer func() {
		go dt.updateTimestamp()
	}()
	for i, col := range dt.columns {
		for j, cell := range col.data {
			if cell == nil {
				dt.columns[i].data[j] = newValue
			} else if cellFloat, ok := cell.(float64); ok {
				if math.IsNaN(cellFloat) {
					dt.columns[i].data[j] = newValue
				}
			}
		}
	}
}

func (dt *DataTable) replaceInRow_notAtomic(rowIndex int, oldValue, newValue any, mode ...int) error {
	defer func() {
		go dt.updateTimestamp()
	}()
	modeFlag := 0
	if len(mode) > 1 {
		return fmt.Errorf("mode parameter can only have 0 or 1 value")
	}
	if len(mode) == 1 {
		modeFlag = mode[0]
	}

	isOldValueNaN := false
	if val, ok := oldValue.(float64); ok && math.IsNaN(val) {
		isOldValueNaN = true
	}

	switch modeFlag {
	case 1:
		// 取代第一個
		for _, col := range dt.columns {
			if rowIndex >= 0 && rowIndex < len(col.data) {
				if isOldValueNaN {
					if val, ok := col.data[rowIndex].(float64); ok && math.IsNaN(val) {
						col.data[rowIndex] = newValue
						go col.updateTimestamp()
						break
					}
				} else if col.data[rowIndex] == oldValue {
					col.data[rowIndex] = newValue
					go col.updateTimestamp()
					break
				}
			}
		}
	case 0: // 取代所有符合的
		for _, col := range dt.columns {
			if rowIndex >= 0 && rowIndex < len(col.data) {
				if isOldValueNaN {
					if val, ok := col.data[rowIndex].(float64); ok && math.IsNaN(val) {
						col.data[rowIndex] = newValue
						go col.updateTimestamp()
					}
				} else if col.data[rowIndex] == oldValue {
					col.data[rowIndex] = newValue
					go col.updateTimestamp()
				}
			}
		}
	case -1: // 從後往前取代第一個
		for i := len(dt.columns) - 1; i >= 0; i-- {
			col := dt.columns[i]
			if rowIndex >= 0 && rowIndex < len(col.data) {
				if isOldValueNaN {
					if val, ok := col.data[rowIndex].(float64); ok && math.IsNaN(val) {
						col.data[rowIndex] = newValue
						go col.updateTimestamp()
						break
					}
				} else if col.data[rowIndex] == oldValue {
					col.data[rowIndex] = newValue
					go col.updateTimestamp()
					break
				}
			}
		}
	default:
		return fmt.Errorf("invalid mode parameter, no replacements made")
	}
	return nil
}

func (dt *DataTable) replaceNaNsAndNilsInRow_notAtomic(rowIndex int, newValue any, mode ...int) error {
	defer func() {
		go dt.updateTimestamp()
	}()
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
					go col.updateTimestamp()
					break
				} else if val, ok := col.data[rowIndex].(float64); ok {
					if math.IsNaN(val) {
						col.data[rowIndex] = newValue
						go col.updateTimestamp()
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
					go col.updateTimestamp()
				} else if val, ok := col.data[rowIndex].(float64); ok {
					if math.IsNaN(val) {
						col.data[rowIndex] = newValue
						go col.updateTimestamp()
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
					go col.updateTimestamp()
					break
				} else if val, ok := col.data[rowIndex].(float64); ok {
					if math.IsNaN(val) {
						col.data[rowIndex] = newValue
						go col.updateTimestamp()
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

func (dt *DataTable) replaceInCol_notAtomic(colIndex string, oldValue, newValue any, mode ...int) error {
	modeFlag := 0
	if len(mode) > 1 {
		return fmt.Errorf("mode parameter can only have 0 or 1 value")
	}
	if len(mode) == 1 {
		modeFlag = mode[0]
	}
	if colNo, ok := utils.ParseColIndex(colIndex); ok && colNo >= 0 && colNo < len(dt.columns) {
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
	} else {
		return fmt.Errorf("column '%s' does not exist", colIndex)
	}
	return nil
}

func (dt *DataTable) replaceNaNsAndNilsInCol_notAtomic(colIndex string, newValue any, mode ...int) error {
	modeFlag := 0
	if len(mode) > 1 {
		return fmt.Errorf("mode parameter can only have 0 or 1 value")
	}
	if len(mode) == 1 {
		modeFlag = mode[0]
	}
	if colNo, ok := utils.ParseColIndex(colIndex); ok && colNo >= 0 && colNo < len(dt.columns) {
		switch modeFlag {
		case 1:
			// 取代第一個
			for i, val := range dt.columns[colNo].data {
				if val == nil {
					dt.columns[colNo].data[i] = newValue
					go dt.columns[colNo].updateTimestamp()
					break
				} else if v, ok := val.(float64); ok && math.IsNaN(v) {
					dt.columns[colNo].data[i] = newValue
					go dt.columns[colNo].updateTimestamp()
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
					go dt.columns[colNo].updateTimestamp()
					break
				} else if v, ok := val.(float64); ok && math.IsNaN(v) {
					dt.columns[colNo].data[i] = newValue
					go dt.columns[colNo].updateTimestamp()
					break
				}
			}
		default:
			return fmt.Errorf("invalid mode parameter, no replacements made")
		}
	} else {
		return fmt.Errorf("column '%s' does not exist", colIndex)
	}
	return nil
}
