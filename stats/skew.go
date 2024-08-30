package stats

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
)

// Skew calculates the skewness(sample) of the DataList.
// Returns the skewness.
// Returns nil if the DataList is empty or the skewness cannot be calculated.
// 錯誤！
func Skew(dl insyra.IDataList, method ...string) interface{} {
	methodStr := "pearson"
	if len(method) > 0 {
		methodStr = method[0]
	}
	if len(method) > 1 {
		fmt.Println("[insyra] DataList.Skew(): Too many arguments, returning nil.")
		return nil
	}
	if dl.Len() == 0 {
		fmt.Println("[insyra] DataList.Skew(): DataList is empty, returning nil.")
		return nil
	}
	data := dl.ToF64Slice()

	var result interface{}
	switch methodStr {
	case "pearson":
		result = calculateSkewPearson(data)
		goto returnResult
	case "moments":
		result = calculateSkewMoments(data)
		goto returnResult
	default:
		fmt.Println("[insyra] DataList.Skew(): Invalid method, returning nil.")
		return nil
	}
returnResult:
	if result == nil {
		fmt.Println("[insyra] DataList.Skew(): Skewness calculation failed, returning nil.")
		return nil
	}
	resultFloat, ok := result.(float64)
	if !ok {
		fmt.Println("[insyra] DataList.Skew(): Skewness is not a float64, returning nil.")
		return nil
	}
	return resultFloat
}

// ======================== calculation functions ========================
func calculateSkewPearson(data []float64) interface{} {
	// todo
	var result float64
	return result
}

func calculateSkewMoments(data []float64) interface{} {
	// todo
	var result float64
	return result
}
