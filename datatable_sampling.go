package insyra

import (
	"math/rand/v2"
)

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
	for _, colName := range colNames {
		ndt.AppendCols(NewDataList().SetName(colName))
	}
	ndt.AppendRowsFromDataList(sampledRows...)

	return ndt
}
