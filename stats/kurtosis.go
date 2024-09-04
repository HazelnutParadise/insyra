package stats

import "github.com/HazelnutParadise/insyra"

// Kurtosis calculates the kurtosis(sample) of the DataList.
// Returns the kurtosis.
// Returns nil if the DataList is empty or the kurtosis cannot be calculated.
// 錯誤！
func Kurtosis(data) interface{} {
	d, dLen := insyra.ProcessData(data)
	d64 := insyra.SliceToF64(d)
}
