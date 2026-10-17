package insyra

import "math"

// IsEqualTo reports whether another DataTable has the same number of rows and
// columns and equal cells: types must match, as in DataList.IsEqualTo, and two
// NaN cells are equal. A cell past the end of a short column
// counts as nil. Table, column and row names are not compared; IsTheSameAs
// compares them too.
func (dt *DataTable) IsEqualTo(other *DataTable) bool {
	return dt.compareTo("IsEqualTo", other, false)
}

// IsTheSameAs reports whether another DataTable is equal by IsEqualTo and has
// the same table name, the same column names in order and the same row names
// in order. A row without a name matches only a row without a name. Use
// a == b to ask whether two values are the same object.
func (dt *DataTable) IsTheSameAs(other *DataTable) bool {
	return dt.compareTo("IsTheSameAs", other, true)
}

func (dt *DataTable) compareTo(funcName string, other *DataTable, withNames bool) bool {
	if dt == other {
		return true
	}
	if other == nil {
		dt.warn(funcName, "Another DataTable is nil, returning false")
		return false
	}
	var result bool
	AtomicDoAll(func() {
		result = sameTableContents(dt, other, withNames)
	}, dt, other)
	return result
}

// sameTableContents compares two tables whose locks the caller holds.
func sameTableContents(a, b *DataTable, withNames bool) bool {
	if len(a.columns) != len(b.columns) {
		return false
	}
	rows := a.getMaxColLength()
	if rows != b.getMaxColLength() {
		return false
	}
	if withNames && a.name != b.name {
		return false
	}
	for c, colA := range a.columns {
		colB := b.columns[c]
		if withNames && colA.name != colB.name {
			return false
		}
		for r := range rows {
			if !equalTableCell(cellAt(colA, r), cellAt(colB, r)) {
				return false
			}
		}
	}
	if withNames {
		for r := range rows {
			nameA, okA := a.getRowNameByIndex(r)
			nameB, okB := b.getRowNameByIndex(r)
			if okA != okB || nameA != nameB {
				return false
			}
		}
	}
	return true
}

// cellAt reads row r of a column, or nil past its end.
func cellAt(col *DataList, r int) any {
	if r < len(col.data) {
		return col.data[r]
	}
	return nil
}

// equalTableCell is DataList.IsEqualTo's cell rule, except that two NaN cells
// are equal, so a table holding a missing float equals its own copy.
func equalTableCell(a, b any) bool {
	if fa, ok := a.(float64); ok {
		if fb, ok := b.(float64); ok && math.IsNaN(fa) && math.IsNaN(fb) {
			return true
		}
	}
	return comparableEqual(a, b)
}
