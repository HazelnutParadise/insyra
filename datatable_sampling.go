package insyra

import (
	"math/rand/v2"
)

// SimpleRandomSample returns a new DataTable containing a simple random sample of the specified size.
// If sampleSize is greater than the number of rows in the DataTable, it returns a copy of the original DataTable.
// If sampleSize is less than or equal to 0, it returns an empty DataTable.
func (dt *DataTable) SimpleRandomSample(sampleSize int) *DataTable {
	if sampleSize <= 0 {
		LogWarning("DataTable", "SimpleRandomSample", "Sample size is less than or equal to 0. Returning an empty DataTable.")
		return NewDataTable()
	}
	var colNames []string
	sampledRows := []*DataList{}
	ndt := NewDataTable()
	var earlyResult *DataTable
	dt.AtomicDo(func(table *DataTable) {
		numRows, _ := table.Size()
		if sampleSize >= numRows {
			LogWarning("DataTable", "SimpleRandomSample", "Sample size is greater than or equal to the number of rows. Returning a copy of the original DataTable.")
			earlyResult = table.Clone()
			return
		}
		indices := rand.Perm(numRows)[:sampleSize]
		colNames = table.ColNames()

		ndt.SetName(table.GetName() + "_Sampled")
		for _, idx := range indices {
			sampledRows = append(sampledRows, table.GetRow(idx))
		}
	})
	if earlyResult != nil {
		return earlyResult
	}
	ndt.SetColNames(colNames)
	ndt.AppendRowsFromDataList(sampledRows...)

	return ndt
}
