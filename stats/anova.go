package stats

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parallel"
	"gonum.org/v1/gonum/mat"
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
	MSB     float64 // Between-group Mean Square
	MSW     float64 // Within-group Mean Square
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
		totalSum += group.Sum()
		totalCount += group.Len()
	}
	totalMean := totalSum / float64(totalCount)

	var SSB, SSW float64

	// 並行計算 SSB 和 SSW
	parallel.GroupUp(
		func() {
			SSB = 0.0
			for _, group := range groups {
				groupMean := group.Mean()
				SSB += float64(group.Len()) * math.Pow(groupMean-totalMean, 2)
			}
		},
		func() {
			SSW = 0.0
			for _, group := range groups {
				groupMean := group.Mean()
				for i := 0; i < group.Len(); i++ {
					value := conv.ParseF64(group.Get(i))
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
		MSB:     SSB / float64(DFB),
		MSW:     SSW / float64(DFW),
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
	MSA      float64 // Mean Square for Factor A
	MSB      float64 // Mean Square for Factor B
	MSAB     float64 // Mean Square for interaction
}

// TwoWayANOVA calculates the two-way ANOVA of the given data table.
// Use wide data format to calculate the ANOVA.
// It returns a pointer to a TwoWayANOVAResult struct containing the results.
func TwoWayANOVA_WideFormat(dataTable insyra.IDataTable) *TwoWayANOVAResult {
	rowNum, colNum := dataTable.Size()

	// 將數據轉換為矩陣格式
	data := mat.NewDense(rowNum, colNum, nil)
	for i := 0; i < rowNum; i++ {
		for j := 0; j < colNum; j++ {
			data.Set(i, j, conv.ParseF64(dataTable.GetElementByNumberIndex(i, j)))
		}
	}

	// 計算總平方和 (Total SS)
	var totalSS float64
	var grandTotal float64
	for i := 0; i < rowNum; i++ {
		for j := 0; j < colNum; j++ {
			grandTotal += data.At(i, j)
		}
	}
	grandMean := grandTotal / float64(rowNum*colNum)

	for i := 0; i < rowNum; i++ {
		for j := 0; j < colNum; j++ {
			totalSS += math.Pow(data.At(i, j)-grandMean, 2)
		}
	}

	// 計算 Factor A (行) 的平方和
	var SSA float64
	rowMeans := make([]float64, rowNum)
	for i := 0; i < rowNum; i++ {
		rowSum := 0.0
		for j := 0; j < colNum; j++ {
			rowSum += data.At(i, j)
		}
		rowMeans[i] = rowSum / float64(colNum)
		SSA += float64(colNum) * math.Pow(rowMeans[i]-grandMean, 2)
	}

	// 計算 Factor B (列) 的平方和
	var SSB float64
	colMeans := make([]float64, colNum)
	for j := 0; j < colNum; j++ {
		colSum := 0.0
		for i := 0; i < rowNum; i++ {
			colSum += data.At(i, j)
		}
		colMeans[j] = colSum / float64(rowNum)
		SSB += float64(rowNum) * math.Pow(colMeans[j]-grandMean, 2)
	}

	// 計算交互作用平方和
	var SSAB float64
	cellMeans := make([][]float64, rowNum)
	for i := range cellMeans {
		cellMeans[i] = make([]float64, colNum)
		for j := range cellMeans[i] {
			cellMeans[i][j] = data.At(i, j)
		}
	}

	for i := 0; i < rowNum; i++ {
		for j := 0; j < colNum; j++ {
			expected := rowMeans[i] + colMeans[j] - grandMean
			SSAB += math.Pow(cellMeans[i][j]-expected, 2)
		}
	}

	// 計算誤差平方和
	SSW := totalSS - SSA - SSB - SSAB

	// 計算自由度
	DFA := rowNum - 1
	DFB := colNum - 1
	DFAxB := DFA * DFB
	DFW := (rowNum * colNum) - (rowNum + colNum - 1)

	// 計算均方
	MSA := SSA / float64(DFA)
	MSB := SSB / float64(DFB)
	MSAB := SSAB / float64(DFAxB)
	MSW := SSW / float64(DFW)

	// 計算 F 值
	FAValue := MSA / MSW
	FBValue := MSB / MSW
	FABValue := MSAB / MSW

	// 安全計算 P 值
	var PAValue, PBValue, PABValue float64

	if DFA > 0 && DFW > 0 && !math.IsNaN(FAValue) && !math.IsInf(FAValue, 0) {
		fDistA := distuv.F{D1: float64(DFA), D2: float64(DFW)}
		PAValue = 1 - fDistA.CDF(FAValue)
	}

	if DFB > 0 && DFW > 0 && !math.IsNaN(FBValue) && !math.IsInf(FBValue, 0) {
		fDistB := distuv.F{D1: float64(DFB), D2: float64(DFW)}
		PBValue = 1 - fDistB.CDF(FBValue)
	}

	if DFAxB > 0 && DFW > 0 && !math.IsNaN(FABValue) && !math.IsInf(FABValue, 0) {
		fDistAB := distuv.F{D1: float64(DFAxB), D2: float64(DFW)}
		PABValue = 1 - fDistAB.CDF(FABValue)
	}

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
		TotalSS:  totalSS,
		MSA:      MSA,
		MSB:      MSB,
		MSAB:     MSAB,
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
	MSB     float64 // Between-group Mean Square
	MSW     float64 // Within-group Mean Square
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
			grandTotal += conv.ParseF64(dataTable.GetElementByNumberIndex(i, j))
		}
	}

	grandMean := grandTotal / float64(rowNum*colNum)

	var ssTotal, ssBetween, ssSubjects float64
	// Calculate SSTotal, SSBetween, and SSSubjects in parallel
	ssTotalFunc := func() {
		ssTotal = 0.0
		for i := 0; i < rowNum; i++ {
			for j := 0; j < colNum; j++ {
				value := conv.ParseF64(dataTable.GetElementByNumberIndex(i, j))
				ssTotal += math.Pow(value-grandMean, 2)
			}
		}
	}

	ssBetweenFunc := func() {
		ssBetween = 0.0
		for i := 0; i < rowNum; i++ {
			conditionMean := dataTable.GetRow(i).Mean()
			ssBetween += float64(colNum) * math.Pow(conditionMean-grandMean, 2)
		}
	}

	ssSubjectsFunc := func() {
		ssSubjects = 0.0
		for j := 0; j < colNum; j++ {
			subjectMean := 0.0
			for i := 0; i < rowNum; i++ {
				subjectMean += conv.ParseF64(dataTable.GetElementByNumberIndex(i, j))
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
		MSB:     MSBetween,
		MSW:     MSWithin,
	}
}

// OneWayANOVA_LongFormat 計算長資料格式的單因子變異數分析
// valueList: 觀察值列表 (IDataList)
// groupList: 對應的組別標識列表 (IDataList)
func OneWayANOVA_LongFormat(valueList, groupList insyra.IDataList) *OneWayANOVAResult {
	if valueList.Len() != groupList.Len() {
		fmt.Println("Data and groups must have the same length")
		return nil
	}

	// 轉換資料為 float64 和 int
	data := make([]float64, valueList.Len())
	groups := make([]int, groupList.Len())

	for i := 0; i < valueList.Len(); i++ {
		data[i] = conv.ParseF64(valueList.Get(i))
		groups[i] = conv.ParseInt(groupList.Get(i))
	}

	// 計算每個組的數據
	groupData := make(map[int][]float64)
	for i, group := range groups {
		groupData[group] = append(groupData[group], data[i])
	}

	// 計算總均值和總數
	totalSum := 0.0
	totalCount := len(data)
	for _, value := range data {
		totalSum += value
	}
	totalMean := totalSum / float64(totalCount)

	var SSB, SSW float64

	// 並行計算 SSB 和 SSW
	parallel.GroupUp(
		func() {
			SSB = 0.0
			for _, values := range groupData {
				groupMean := 0.0
				for _, v := range values {
					groupMean += v
				}
				groupMean /= float64(len(values))
				SSB += float64(len(values)) * math.Pow(groupMean-totalMean, 2)
			}
		},
		func() {
			SSW = 0.0
			for _, values := range groupData {
				groupMean := 0.0
				for _, v := range values {
					groupMean += v
				}
				groupMean /= float64(len(values))

				for _, v := range values {
					SSW += math.Pow(v-groupMean, 2)
				}
			}
		},
	).Run().AwaitResult()

	// 計算自由度
	DFB := len(groupData) - 1
	DFW := totalCount - len(groupData)

	// 如果自由度無效，直接返回 nil
	if DFB <= 0 || DFW <= 0 {
		fmt.Println("Degrees of Freedom must be greater than 0")
		return nil
	}

	// 計算 F 值
	FValue := (SSB / float64(DFB)) / (SSW / float64(DFW))

	// 計算 P 值
	fDist := distuv.F{
		D1: float64(DFB),
		D2: float64(DFW),
	}
	PValue := 1 - fDist.CDF(FValue)

	return &OneWayANOVAResult{
		SSB:     SSB,
		SSW:     SSW,
		FValue:  FValue,
		PValue:  PValue,
		DFB:     DFB,
		DFW:     DFW,
		TotalSS: SSB + SSW,
		MSB:     SSB / float64(DFB),
		MSW:     SSW / float64(DFW),
	}
}

// TwoWayANOVA_LongFormat 計算長資料格式的雙因子變異數分析
// valueList: 觀察值列表 (IDataList)
// factorAList: 第一個因子的組別標識列表 (IDataList)
// factorBList: 第二個因子的組別標識列表 (IDataList)
func TwoWayANOVA_LongFormat(valueList, factorAList, factorBList insyra.IDataList) *TwoWayANOVAResult {
	if valueList.Len() != factorAList.Len() || valueList.Len() != factorBList.Len() {
		fmt.Println("Data and factors must have the same length")
		return nil
	}

	// 轉換資料
	N := valueList.Len()
	values := make([]float64, N)
	factorsA := make([]int, N)
	factorsB := make([]int, N)

	for i := 0; i < N; i++ {
		values[i] = conv.ParseF64(valueList.Get(i))
		factorsA[i] = conv.ParseInt(factorAList.Get(i))
		factorsB[i] = conv.ParseInt(factorBList.Get(i))
	}

	// 獲取因子水平數
	levelsA := make(map[int]bool)
	levelsB := make(map[int]bool)
	for i := 0; i < N; i++ {
		levelsA[factorsA[i]] = true
		levelsB[factorsB[i]] = true
	}
	a := len(levelsA) // Factor A 水平數
	b := len(levelsB) // Factor B 水平數

	// 計算總和和均值
	totalSum := 0.0
	for _, v := range values {
		totalSum += v
	}
	grandMean := totalSum / float64(N)

	// 計算 A 因子的效應
	factorASums := make(map[int]float64)
	factorACounts := make(map[int]int)
	for i := 0; i < N; i++ {
		factorASums[factorsA[i]] += values[i]
		factorACounts[factorsA[i]]++
	}

	SSA := 0.0
	for level, sum := range factorASums {
		mean := sum / float64(factorACounts[level])
		SSA += float64(factorACounts[level]) * math.Pow(mean-grandMean, 2)
	}

	// 計算 B 因子的效應
	factorBSums := make(map[int]float64)
	factorBCounts := make(map[int]int)
	for i := 0; i < N; i++ {
		factorBSums[factorsB[i]] += values[i]
		factorBCounts[factorsB[i]]++
	}

	SSB := 0.0
	for level, sum := range factorBSums {
		mean := sum / float64(factorBCounts[level])
		SSB += float64(factorBCounts[level]) * math.Pow(mean-grandMean, 2)
	}

	// 計算交互作用
	cellSums := make(map[string]float64)
	cellCounts := make(map[string]int)
	for i := 0; i < N; i++ {
		key := fmt.Sprintf("%d_%d", factorsA[i], factorsB[i])
		cellSums[key] += values[i]
		cellCounts[key]++
	}

	SSAB := 0.0
	for key, sum := range cellSums {
		parts := strings.Split(key, "_")
		levelA, _ := strconv.Atoi(parts[0])
		levelB, _ := strconv.Atoi(parts[1])

		cellMean := sum / float64(cellCounts[key])
		meanA := factorASums[levelA] / float64(factorACounts[levelA])
		meanB := factorBSums[levelB] / float64(factorBCounts[levelB])

		expected := meanA + meanB - grandMean
		SSAB += float64(cellCounts[key]) * math.Pow(cellMean-expected, 2)
	}

	// 計算總平方和
	TSS := 0.0
	for _, value := range values {
		TSS += math.Pow(value-grandMean, 2)
	}

	// 計算誤差平方和
	SSE := TSS - SSA - SSB - SSAB

	// 計算自由度
	DFA := a - 1
	DFB := b - 1
	DFAxB := DFA * DFB
	DFE := N - (a * b)

	// 計算均方
	MSA := SSA / float64(DFA)
	MSB := SSB / float64(DFB)
	MSAB := SSAB / float64(DFAxB)
	MSW := SSE / float64(DFE)

	// 計算 F 值
	FA := MSA / MSW
	FB := MSB / MSW
	FAB := MSAB / MSW

	// 計算 P 值
	var PA, PB, PAB float64
	if DFA > 0 && DFE > 0 && !math.IsNaN(FA) && !math.IsInf(FA, 0) {
		fDistA := distuv.F{D1: float64(DFA), D2: float64(DFE)}
		PA = 1 - fDistA.CDF(FA)
	}
	if DFB > 0 && DFE > 0 && !math.IsNaN(FB) && !math.IsInf(FB, 0) {
		fDistB := distuv.F{D1: float64(DFB), D2: float64(DFE)}
		PB = 1 - fDistB.CDF(FB)
	}
	if DFAxB > 0 && DFE > 0 && !math.IsNaN(FAB) && !math.IsInf(FAB, 0) {
		fDistAB := distuv.F{D1: float64(DFAxB), D2: float64(DFE)}
		PAB = 1 - fDistAB.CDF(FAB)
	}

	return &TwoWayANOVAResult{
		SSA:      SSA,
		SSB:      SSB,
		SSAB:     SSAB,
		SSW:      SSE,
		FAValue:  FA,
		FBValue:  FB,
		FABValue: FAB,
		PAValue:  PA,
		PBValue:  PB,
		PABValue: PAB,
		DFA:      DFA,
		DFB:      DFB,
		DFAxB:    DFAxB,
		DFW:      DFE,
		TotalSS:  TSS,
		MSA:      MSA,
		MSB:      MSB,
		MSAB:     MSAB,
	}
}

// RepeatedMeasuresANOVA_LongFormat 計算長資料格式的重複測量變異數分析
// valueList: 觀察值列表 (IDataList)
// subjectList: 受試者編號列表 (IDataList)
// conditionList: 實驗條件編號列表 (IDataList)
func RepeatedMeasuresANOVA_LongFormat(valueList, subjectList, conditionList insyra.IDataList) *RepeatedMeasuresANOVAResult {
	if valueList.Len() != subjectList.Len() || valueList.Len() != conditionList.Len() {
		fmt.Println("Data, subjects, and conditions must have the same length")
		return nil
	}

	// 轉換資料為 float64 和 int
	data := make([]float64, valueList.Len())
	subjects := make([]int, subjectList.Len())
	conditions := make([]int, conditionList.Len())

	for i := 0; i < valueList.Len(); i++ {
		data[i] = conv.ParseF64(valueList.Get(i))
		subjects[i] = conv.ParseInt(subjectList.Get(i))
		conditions[i] = conv.ParseInt(conditionList.Get(i))
	}

	// 計算總均值
	totalSum := 0.0
	for _, value := range data {
		totalSum += value
	}
	totalCount := len(data)
	grandMean := totalSum / float64(totalCount)

	// 獲取不同受試者和條件的數量
	uniqueSubjects := make(map[int]bool)
	uniqueConditions := make(map[int]bool)
	for i := 0; i < len(data); i++ {
		uniqueSubjects[subjects[i]] = true
		uniqueConditions[conditions[i]] = true
	}
	numSubjects := len(uniqueSubjects)
	numConditions := len(uniqueConditions)

	var ssTotal, ssBetween, ssSubjects float64

	// 並行計算各種平方和
	parallel.GroupUp(
		func() {
			// 計算總平方和
			ssTotal = 0.0
			for _, value := range data {
				ssTotal += math.Pow(value-grandMean, 2)
			}
		},
		func() {
			// 計算組間平方和（條件間）
			conditionMeans := make(map[int]float64)
			conditionCounts := make(map[int]int)
			for i, value := range data {
				conditionMeans[conditions[i]] += value
				conditionCounts[conditions[i]]++
			}

			ssBetween = 0.0
			for condition, sum := range conditionMeans {
				mean := sum / float64(conditionCounts[condition])
				ssBetween += float64(conditionCounts[condition]) * math.Pow(mean-grandMean, 2)
			}
		},
		func() {
			// 計算受試者平方和
			subjectMeans := make(map[int]float64)
			subjectCounts := make(map[int]int)
			for i, value := range data {
				subjectMeans[subjects[i]] += value
				subjectCounts[subjects[i]]++
			}

			ssSubjects = 0.0
			for subject, sum := range subjectMeans {
				mean := sum / float64(subjectCounts[subject])
				ssSubjects += float64(subjectCounts[subject]) * math.Pow(mean-grandMean, 2)
			}
		},
	).Run().AwaitResult()

	// 計算組內平方和
	SSWithin := ssTotal - ssBetween - ssSubjects

	// 計算自由度
	DFBetween := numConditions - 1
	DFSubjects := numSubjects - 1
	DFWithin := (numConditions - 1) * (numSubjects - 1)

	// 如果自由度無效，直接返回 nil
	if DFBetween <= 0 || DFWithin <= 0 {
		fmt.Println("Degrees of Freedom must be greater than 0")
		return nil
	}

	// 計算均方
	MSBetween := ssBetween / float64(DFBetween)
	MSWithin := SSWithin / float64(DFWithin)

	// 計算 F 值
	FValue := MSBetween / MSWithin

	// 計算 P 值
	fDist := distuv.F{
		D1: float64(DFBetween),
		D2: float64(DFWithin),
	}
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
		MSB:     MSBetween,
		MSW:     MSWithin,
	}
}
