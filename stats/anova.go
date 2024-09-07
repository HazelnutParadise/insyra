package stats

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

type ANOVAResult struct {
	SSB     float64 // Between-group Sum of Squares
	SSW     float64 // Within-group Sum of Squares
	FValue  float64 // F-value
	PValue  float64 // P-value
	DFB     int     // Between-group Degrees of Freedom
	DFW     int     // Within-group Degrees of Freedom
	TotalSS float64 // Total Sum of Squares
}

// OneWayANOVA 接受 DataTable 並執行單因子 ANOVA 分析。
func OneWayANOVA(dataTable insyra.IDataTable) *ANOVAResult {
	var groups []insyra.IDataList

	// 將每一列資料視為不同的組別
	colNum, _ := dataTable.Size()
	for i := 0; i < colNum; i++ {
		column := dataTable.GetColumnByNumber(i)
		groups = append(groups, column)
	}

	// 計算總均值
	totalSum := 0.0
	totalCount := 0
	for _, group := range groups {
		totalSum += group.Sum().(float64)
		totalCount += group.Len()
	}
	totalMean := totalSum / float64(totalCount)

	// 計算 SSB (Between-group Sum of Squares)
	SSB := 0.0
	for _, group := range groups {
		groupMean := group.Mean().(float64)
		SSB += float64(group.Len()) * math.Pow(groupMean-totalMean, 2)
	}

	// 計算 SSW (Within-group Sum of Squares)
	SSW := 0.0
	for _, group := range groups {
		groupMean := group.Mean().(float64)
		for i := 0; i < group.Len(); i++ {
			value, _ := group.Get(i).(float64)
			SSW += math.Pow(value-groupMean, 2)
		}
	}

	// 計算自由度
	DFB := len(groups) - 1          // Between-group Degrees of Freedom
	DFW := totalCount - len(groups) // Within-group Degrees of Freedom

	// 如果自由度無效，直接返回 nil
	if DFB <= 0 || DFW <= 0 {
		fmt.Println("Degrees of Freedom must be greater than 0")
		return nil
	}

	// 計算 F 值
	FValue := (SSB / float64(DFB)) / (SSW / float64(DFW))

	// 使用 F 分佈計算 P 值
	fDist := distuv.F{
		D1: float64(DFB), // Between-group Degrees of Freedom
		D2: float64(DFW), // Within-group Degrees of Freedom
	}
	PValue := 1 - fDist.CDF(FValue)

	// 返回結果
	return &ANOVAResult{
		SSB:     SSB,
		SSW:     SSW,
		FValue:  FValue,
		PValue:  PValue,
		DFB:     DFB,
		DFW:     DFW,
		TotalSS: SSB + SSW,
	}
}
