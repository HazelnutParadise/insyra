package stats

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parallel"
	"gonum.org/v1/gonum/stat/distuv"
)

type OneWayANOVAResult struct {
	SSB     float64 // Between-group Sum of Squares
	SSW     float64 // Within-group Sum of Squares
	FValue  float64 // F-value
	PValue  float64 // P-value
	DFB     int     // Between-group Degrees of Freedom
	DFW     int     // Within-group Degrees of Freedom
	TotalSS float64 // Total Sum of Squares
}

func OneWayANOVA_WideFormat(dataTable insyra.IDataTable) *OneWayANOVAResult {
	var groups []insyra.IDataList

	// 將每一行資料視為不同的組別
	rowNum, _ := dataTable.Size()
	for i := 0; i < rowNum; i++ {
		row := dataTable.GetRow(i)
		groups = append(groups, row)
	}

	// 計算總均值和總數
	totalSum := 0.0
	totalCount := 0
	for _, group := range groups {
		totalSum += group.Sum().(float64)
		totalCount += group.Len()
	}
	totalMean := totalSum / float64(totalCount)

	var SSB, SSW float64

	// 並行計算 SSB 和 SSW
	parallel.GroupUp(
		func() {
			SSB = 0.0
			for _, group := range groups {
				groupMean := group.Mean().(float64)
				SSB += float64(group.Len()) * math.Pow(groupMean-totalMean, 2)
			}
		},
		func() {
			SSW = 0.0
			for _, group := range groups {
				groupMean := group.Mean().(float64)
				for i := 0; i < group.Len(); i++ {
					value, _ := group.Get(i).(float64)
					SSW += math.Pow(value-groupMean, 2)
				}
			}

		},
	).Run().AwaitResult()

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
	return &OneWayANOVAResult{
		SSB:     SSB,
		SSW:     SSW,
		FValue:  FValue,
		PValue:  PValue,
		DFB:     DFB,
		DFW:     DFW,
		TotalSS: SSB + SSW,
	}
}

type TwoWayANOVAResult struct {
	SSA      float64 // Factor A Sum of Squares
	SSB      float64 // Factor B Sum of Squares
	SSAB     float64 // Interaction Sum of Squares
	SSW      float64 // Within-group Sum of Squares
	FAValue  float64 // F-value for Factor A
	FBValue  float64 // F-value for Factor B
	FABValue float64 // F-value for interaction
	PAValue  float64 // P-value for Factor A
	PBValue  float64 // P-value for Factor B
	PABValue float64 // P-value for interaction
	DFA      int     // Degrees of Freedom for Factor A
	DFB      int     // Degrees of Freedom for Factor B
	DFAxB    int     // Degrees of Freedom for interaction
	DFW      int     // Degrees of Freedom within groups
	TotalSS  float64 // Total Sum of Squares
}

// TwoWayANOVA calculates the two-way ANOVA of the given data table.
// Use wide data format to calculate the ANOVA.
// It returns a pointer to a TwoWayANOVAResult struct containing the results.
func TwoWayANOVA_WideFormat(dataTable insyra.IDataTable) *TwoWayANOVAResult {
	var observations []float64
	var factorsA, factorsB []int

	// 將寬資料轉換為長資料格式
	rowNum, colNum := dataTable.Size()
	for i := 0; i < rowNum; i++ {
		for j := 0; j < colNum; j++ {
			observations = append(observations, dataTable.GetElementByNumberIndex(i, j).(float64))
			factorsA = append(factorsA, i)
			factorsB = append(factorsB, j)
		}
	}

	// 計算總和和均值
	totalSum := 0.0
	for _, value := range observations {
		totalSum += value
	}
	totalCount := len(observations)
	totalMean := totalSum / float64(totalCount)

	// 並行計算 SSA 和 SSB
	var SSA, SSB float64
	calcSS := func() {
		for i := 0; i < rowNum; i++ {
			groupSum := 0.0
			count := 0
			for j, factor := range factorsA {
				if factor == i {
					groupSum += observations[j]
					count++
				}
			}
			groupMean := groupSum / float64(count)
			SSA += float64(count) * math.Pow(groupMean-totalMean, 2)
		}
	}

	calcSSB := func() {
		for j := 0; j < colNum; j++ {
			groupSum := 0.0
			count := 0
			for i, factor := range factorsB {
				if factor == j {
					groupSum += observations[i]
					count++
				}
			}
			groupMean := groupSum / float64(count)
			SSB += float64(count) * math.Pow(groupMean-totalMean, 2)
		}
	}

	parallel.GroupUp(calcSS, calcSSB).Run().AwaitResult()

	// 計算 SSAB 和 SSW
	var SSAB, SSW float64
	for idx, value := range observations {
		meanA := 0.0
		meanB := 0.0

		for i := 0; i < rowNum; i++ {
			if factorsA[idx] == i {
				groupSum := 0.0
				count := 0
				for j, factor := range factorsA {
					if factor == i {
						groupSum += observations[j]
						count++
					}
				}
				meanA = groupSum / float64(count)
				break
			}
		}

		for j := 0; j < colNum; j++ {
			if factorsB[idx] == j {
				groupSum := 0.0
				count := 0
				for i, factor := range factorsB {
					if factor == j {
						groupSum += observations[i]
						count++
					}
				}
				meanB = groupSum / float64(count)
				break
			}
		}

		expected := meanA + meanB - totalMean
		SSAB += math.Pow(value-expected, 2)
		SSW += math.Pow(value-expected, 2)
	}

	// 計算自由度
	DFA := rowNum - 1
	DFB := colNum - 1
	DFAxB := DFA * DFB
	DFW := totalCount - rowNum - colNum + 1

	// 計算 F 值
	FAValue := (SSA / float64(DFA)) / (SSW / float64(DFW))
	FBValue := (SSB / float64(DFB)) / (SSW / float64(DFW))
	FABValue := (SSAB / float64(DFAxB)) / (SSW / float64(DFW))

	// 並行計算 P 值
	var PAValue, PBValue, PABValue float64
	calcPA := func() {
		fDistA := distuv.F{D1: float64(DFA), D2: float64(DFW)}
		PAValue = 1 - fDistA.CDF(FAValue)
	}

	calcPB := func() {
		fDistB := distuv.F{D1: float64(DFB), D2: float64(DFW)}
		PBValue = 1 - fDistB.CDF(FBValue)
	}

	calcPAB := func() {
		fDistAB := distuv.F{D1: float64(DFAxB), D2: float64(DFW)}
		PABValue = 1 - fDistAB.CDF(FABValue)
	}

	parallel.GroupUp(calcPA, calcPB, calcPAB).Run().AwaitResult()

	// 返回結果
	return &TwoWayANOVAResult{
		SSA:      SSA,
		SSB:      SSB,
		SSAB:     SSAB,
		SSW:      SSW,
		FAValue:  FAValue,
		FBValue:  FBValue,
		FABValue: FABValue,
		PAValue:  PAValue,
		PBValue:  PBValue,
		PABValue: PABValue,
		DFA:      DFA,
		DFB:      DFB,
		DFAxB:    DFAxB,
		DFW:      DFW,
		TotalSS:  SSA + SSB + SSAB + SSW,
	}
}

type RepeatedMeasuresANOVAResult struct {
	SSB     float64 // Between-group Sum of Squares
	SSW     float64 // Within-group Sum of Squares
	FValue  float64 // F-value
	PValue  float64 // P-value
	DFB     int     // Between-group Degrees of Freedom
	DFW     int     // Within-group Degrees of Freedom
	DFSubj  int     // Degrees of Freedom for subjects
	TotalSS float64 // Total Sum of Squares
}

// RepeatedMeasuresANOVA_WideFormat calculates the repeated measures ANOVA of the given data table.
// Use wide data format to calculate the ANOVA.
// It returns a pointer to a RepeatedMeasuresANOVAResult struct containing the results.
func RepeatedMeasuresANOVA_WideFormat(dataTable insyra.IDataTable) *RepeatedMeasuresANOVAResult {
	rowNum, colNum := dataTable.Size() // rowNum: number of conditions, colNum: number of subjects

	// Calculate grand mean

	grandTotal := 0.0
	for i := 0; i < rowNum; i++ {
		for j := 0; j < colNum; j++ {
			grandTotal += dataTable.GetElementByNumberIndex(i, j).(float64)
		}
	}

	grandMean := grandTotal / float64(rowNum*colNum)

	var ssTotal, ssBetween, ssSubjects float64
	// Calculate SSTotal, SSBetween, and SSSubjects in parallel
	ssTotalFunc := func() {
		ssTotal = 0.0
		for i := 0; i < rowNum; i++ {
			for j := 0; j < colNum; j++ {
				value := dataTable.GetElementByNumberIndex(i, j).(float64)
				ssTotal += math.Pow(value-grandMean, 2)
			}
		}
	}

	ssBetweenFunc := func() {
		ssBetween = 0.0
		for i := 0; i < rowNum; i++ {
			conditionMean := dataTable.GetRow(i).Mean().(float64)
			ssBetween += float64(colNum) * math.Pow(conditionMean-grandMean, 2)
		}
	}

	ssSubjectsFunc := func() {
		ssSubjects = 0.0
		for j := 0; j < colNum; j++ {
			subjectMean := 0.0
			for i := 0; i < rowNum; i++ {
				subjectMean += dataTable.GetElementByNumberIndex(i, j).(float64)
			}
			subjectMean /= float64(rowNum)
			ssSubjects += float64(rowNum) * math.Pow(subjectMean-grandMean, 2)
		}
	}

	parallel.GroupUp(ssTotalFunc, ssBetweenFunc, ssSubjectsFunc).Run().AwaitResult()

	// Calculate SSWithin
	SSWithin := ssTotal - ssBetween - ssSubjects

	// Calculate degrees of freedom
	DFBetween := rowNum - 1
	DFSubjects := colNum - 1
	DFWithin := (rowNum - 1) * (colNum - 1)

	// Calculate Mean Squares
	MSBetween := ssBetween / float64(DFBetween)
	MSWithin := SSWithin / float64(DFWithin)

	// Calculate F-value
	FValue := MSBetween / MSWithin

	// Calculate p-value
	fDist := distuv.F{D1: float64(DFBetween), D2: float64(DFWithin)}
	PValue := 1 - fDist.CDF(FValue)

	return &RepeatedMeasuresANOVAResult{
		SSB:     ssBetween,
		SSW:     SSWithin,
		FValue:  FValue,
		PValue:  PValue,
		DFB:     DFBetween,
		DFW:     DFWithin,
		DFSubj:  DFSubjects,
		TotalSS: ssTotal,
	}
}
