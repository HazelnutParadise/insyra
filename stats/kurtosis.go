package stats

import "github.com/HazelnutParadise/insyra"

type KurtosisMode int

const (
	// Excess represents excess kurtosis calculation mode.
	Kurt_Excess KurtosisMode = iota

	// Moment represents moment kurtosis calculation mode.
	Kurt_Moment

	// Fisher represents fisher kurtosis calculation mode.
	Kurt_Fisher

	// Sample represents sample kurtosis calculation mode.
	Kurt_Sample

	// SampleExcess represents sample excess kurtosis calculation mode.
	Kurt_SampleExcess
)

// Kurtosis calculates the kurtosis(sample) of the DataList.
// Returns the kurtosis.
// Returns nil if the DataList is empty or the kurtosis cannot be calculated.
// 錯誤！
func Kurtosis(data, method ...KurtosisMode) interface{} {
	d, dLen := insyra.ProcessData(data)
	d64 := insyra.SliceToF64(d)
	usemethod := Kurt_Excess
	if len(method) > 1 {
		LogWarning("stats.Kurtosis(): More than one method specified, returning nil.")
		return nil
	}
	if len(method) == 1 {
		usemethod = method[0]
	}
}
