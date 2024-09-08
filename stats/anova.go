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

	// 將每一列資料視為不同的組別
	colNum, _ := dataTable.Size()
	for i := 0; i < colNum; i++ {
		column := dataTable.GetColumnByNumber(i)
		groups = append(groups, column)
	}

	// 計算總均值和總數
	totalSum := 0.0
	totalCount := 0
	for _, group := range groups {
		totalSum += group.Sum().(float64)
		totalCount += group.Len()
	}
	totalMean := totalSum / float64(totalCount)

	// 並行計算 SSB 和 SSW
	pg := parallel.GroupUp(
		func() (float64, float64) {
			SSB := 0.0
			for _, group := range groups {
				groupMean := group.Mean().(float64)
				SSB += float64(group.Len()) * math.Pow(groupMean-totalMean, 2)
			}
			return SSB, 0
		},
		func() (float64, float64) {
			SSW := 0.0
			for _, group := range groups {
				groupMean := group.Mean().(float64)
				for i := 0; i < group.Len(); i++ {
					value, _ := group.Get(i).(float64)
					SSW += math.Pow(value-groupMean, 2)
				}
			}
			return 0, SSW
		},
	).Run()

	results := pg.AwaitResult()
	SSB := results[0][0].(float64)
	SSW := results[1][1].(float64)

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
// Use long data format to calculate the ANOVA.
// It returns a pointer to a TwoWayANOVAResult struct containing the results.
func TwoWayANOVA(dataTable insyra.IDataTable) *TwoWayANOVAResult {
	var observations []float64
	var factorsA, factorsB []int

	// 將寬資料轉換為長資料格式
	rowNum, colNum := dataTable.Size()
	for i := 0; i < rowNum; i++ {
		for j := 0; j < colNum; j++ {
			// 保存觀測值
			observations = append(observations, dataTable.GetElementByNumberIndex(i, j).(float64))
			// 因子 A 是行
			factorsA = append(factorsA, i)
			// 因子 B 是列
			factorsB = append(factorsB, j)
		}
	}

	// 計算總均值
	totalSum := 0.0
	totalCount := len(observations)
	for _, value := range observations {
		totalSum += value
	}
	totalMean := totalSum / float64(totalCount)

	// 計算 SSA 和 SSB
	SSA, SSB := 0.0, 0.0
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

	// 計算 SSAB 和 SSW
	SSAB, SSW := 0.0, 0.0
	for idx, value := range observations {
		meanA := 0.0
		meanB := 0.0

		// 找出當前觀測值對應的因子 A 和 B 的均值
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

		// 計算交互項和組內變異
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

	// 使用 F 分佈計算 P 值
	fDistA := distuv.F{D1: float64(DFA), D2: float64(DFW)}
	PAValue := 1 - fDistA.CDF(FAValue)

	fDistB := distuv.F{D1: float64(DFB), D2: float64(DFW)}
	PBValue := 1 - fDistB.CDF(FBValue)

	fDistAB := distuv.F{D1: float64(DFAxB), D2: float64(DFW)}
	PABValue := 1 - fDistAB.CDF(FABValue)

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

// type RepeatedMeasuresANOVAResult struct {
// 	SSB     float64 // Between-group Sum of Squares
// 	SSW     float64 // Within-group Sum of Squares
// 	SSSubj  float64 // Sum of Squares for subjects
// 	FValue  float64 // F-value
// 	PValue  float64 // P-value
// 	DFB     int     // Between-group Degrees of Freedom
// 	DFW     int     // Within-group Degrees of Freedom
// 	DFSubj  int     // Degrees of Freedom for subjects
// 	TotalSS float64 // Total Sum of Squares
// }

// func RepeatedMeasuresANOVA(dataTable insyra.IDataTable) *RepeatedMeasuresANOVAResult {
// 	// 使用行代表組別，列代表受試者
// 	var subjects []insyra.IDataList
// 	rowNum, _ := dataTable.Size()
// 	for i := 0; i < rowNum; i++ {
// 		subjects = append(subjects, dataTable.GetRow(i))
// 	}

// 	// 計算總均值和總數
// 	totalSum := 0.0
// 	totalCount := 0
// 	for _, subject := range subjects {
// 		totalSum += subject.Sum().(float64)
// 		totalCount += subject.Len()
// 	}
// 	totalMean := totalSum / float64(totalCount)

// 	// 計算 SSB 和 SSSubj
// 	SSB, SSSubj := 0.0, 0.0
// 	for _, subject := range subjects {
// 		subjMean := subject.Mean().(float64)
// 		SSB += float64(subject.Len()) * math.Pow(subjMean-totalMean, 2)
// 		for i := 0; i < subject.Len(); i++ {
// 			value, _ := subject.Get(i).(float64)
// 			SSSubj += math.Pow(value-subjMean, 2)
// 		}
// 	}

// 	// 計算自由度
// 	DFB := rowNum - 1
// 	DFW := totalCount - rowNum
// 	DFSubj := totalCount / rowNum

// 	// 計算 F 值
// 	FValue := (SSB / float64(DFB)) / (SSSubj / float64(DFSubj))

// 	// 使用 F 分佈計算 P 值
// 	fDist := distuv.F{D1: float64(DFB), D2: float64(DFW)}
// 	PValue := 1 - fDist.CDF(FValue)

// 	// 返回結果
// 	return &RepeatedMeasuresANOVAResult{
// 		SSB:     SSB,
// 		SSW:     SSSubj,
// 		SSSubj:  SSSubj,
// 		FValue:  FValue,
// 		PValue:  PValue,
// 		DFB:     DFB,
// 		DFW:     DFW,
// 		DFSubj:  DFSubj,
// 		TotalSS: SSB + SSSubj,
// 	}
// }
