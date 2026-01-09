package insyra

import "github.com/HazelnutParadise/insyra/internal/utils"

// GetColIndexByName returns the column index (A, B, C, ...) by its name.
func (dt *DataTable) GetColIndexByName(name string) string {
	var result = ""
	var ok bool
	dt.AtomicDo(func(dt *DataTable) {
		result, ok = dt.getColIndexByName_notAtomic(name)
	})
	if !ok {
		dt.warn("GetColIndexByName", "Column name not found: %s, returning empty string", name)
	}
	return result
}

// GetColIndexByNumber returns the column index (A, B, C, ...) by its number (0, 1, 2, ...).
func (dt *DataTable) GetColIndexByNumber(number int) string {
	var result = ""
	dt.AtomicDo(func(dt *DataTable) {
		if number < 0 {
			number += len(dt.columns)
		}
		if number >= 0 && number < len(dt.columns) {
			result, _ = utils.CalcColIndex(number)
		}
	})
	if result == "" {
		dt.warn("GetColIndexByNumber", "Column number not found: %d, returning empty string", number)
	}
	return result
}

// ----------------------- not atomic below ----------------------

func (dt *DataTable) getColIndexByName_notAtomic(name string) (string, bool) {
	num, ok := dt.getColNumberByName_notAtomic(name)
	if !ok {
		return "", false
	}
	if num >= 0 && num < len(dt.columns) {
		return utils.CalcColIndex(num)
	}
	return "", false
}
