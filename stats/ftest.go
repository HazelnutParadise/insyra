package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

type FTestResult struct {
	FValue float64
	PValue float64
	Df1    int
	Df2    int
}

// FTestForVarianceEquality performs an F-test for variance equality
func FTestForVarianceEquality(data1, data2 *insyra.DataList) *FTestResult {
	// 計算方差
	var1 := data1.Var().(float64)
	var2 := data2.Var().(float64)

	// 計算 F 值
	var fValue float64
	if var1 > var2 {
		fValue = var1 / var2
	} else {
		fValue = var2 / var1
	}

	// 計算自由度
	df1 := data1.Len() - 1
	df2 := data2.Len() - 1

	// 使用卡方分佈計算 P 值
	fDist := distuv.F{D1: float64(df1), D2: float64(df2)}
	pValue := 2 * math.Min(fDist.CDF(fValue), 1-fDist.CDF(fValue)) // 使用雙尾檢定

	return &FTestResult{
		FValue: fValue,
		PValue: pValue,
		Df1:    df1,
		Df2:    df2,
	}
}
