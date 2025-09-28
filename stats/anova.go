package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parallel"
	"gonum.org/v1/gonum/stat/distuv"
)

type ANOVAResultComponent struct {
	SumOfSquares float64
	DF           int
	F            float64
	P            float64
	EtaSquared   float64
}

type TwoWayANOVAResult struct {
	FactorA     ANOVAResultComponent
	FactorB     ANOVAResultComponent
	Interaction ANOVAResultComponent
	Within      ANOVAResultComponent
	TotalSS     float64
}

type OneWayANOVAResult struct {
	Factor  ANOVAResultComponent
	Within  ANOVAResultComponent
	TotalSS float64
}

type RepeatedMeasuresANOVAResult struct {
	Factor  ANOVAResultComponent
	Subject ANOVAResultComponent
	Within  ANOVAResultComponent
	TotalSS float64
}

func OneWayANOVA(groups ...insyra.IDataList) *OneWayANOVAResult {
	if len(groups) < 2 {
		insyra.LogWarning("stats", "OneWayANOVA", "At least two groups are required")
		return nil
	}

	totalSum := 0.0
	totalCount := 0
	for i, g := range groups {
		var groupLen int
		var groupSum float64
		g.AtomicDo(func(gdl *insyra.DataList) {
			groupLen = gdl.Len()
			groupSum = gdl.Sum()
		})

		if groupLen == 0 {
			insyra.LogWarning("stats", "OneWayANOVA", "Group %d is empty", i)
			return nil
		}
		totalSum += groupSum
		totalCount += groupLen
	}

	totalMean := totalSum / float64(totalCount)

	var SSB, SSW float64

	parallel.GroupUp(func() {
		for _, g := range groups {
			var groupMean float64
			var groupLen int
			g.AtomicDo(func(gdl *insyra.DataList) {
				groupMean = gdl.Mean()
				groupLen = gdl.Len()
			})
			SSB += float64(groupLen) * (groupMean - totalMean) * (groupMean - totalMean)
		}
	}, func() {
		for i, g := range groups {
			var groupData []any
			var groupMean float64
			g.AtomicDo(func(gdl *insyra.DataList) {
				groupMean = gdl.Mean()
				groupData = gdl.Data()
			})
			for j, v := range groupData {
				x, ok := insyra.ToFloat64Safe(v)
				if !ok {
					insyra.LogWarning("stats", "OneWayANOVA", "Invalid data at group %d index %d", i, j)
					return
				}
				SSW += (x - groupMean) * (x - groupMean)
			}
		}
	}).Run().AwaitResult()

	DFB := len(groups) - 1
	DFW := totalCount - len(groups)
	F := (SSB / float64(DFB)) / (SSW / float64(DFW))
	P := 1 - distuv.F{D1: float64(DFB), D2: float64(DFW)}.CDF(F)

	return &OneWayANOVAResult{
		Factor:  ANOVAResultComponent{SSB, DFB, F, P, SSB / (SSB + SSW)},
		Within:  ANOVAResultComponent{SSW, DFW, math.NaN(), math.NaN(), math.NaN()},
		TotalSS: SSB + SSW,
	}
}

func TwoWayANOVA(factorALevels, factorBLevels int, cells ...insyra.IDataList) *TwoWayANOVAResult {
	if factorALevels < 2 || factorBLevels < 2 || len(cells) != factorALevels*factorBLevels {
		insyra.LogWarning("stats", "TwoWayANOVA", "Invalid levels or cells")
		return nil
	}

	var allValues []float64
	cellCounts := make([]int, len(cells))
	factorsA := make([]int, 0)
	factorsB := make([]int, 0)

	for i := range factorALevels {
		for j := range factorBLevels {
			cell := cells[i*factorBLevels+j]
			var cellData []any
			var cellLen int
			cell.AtomicDo(func(cdl *insyra.DataList) {
				cellData = cdl.Data()
				cellLen = cdl.Len()
			})
			if cellLen == 0 {
				insyra.LogWarning("stats", "TwoWayANOVA", "Empty cell")
				return nil
			}
			cellCounts[i*factorBLevels+j] = cellLen
			for _, v := range cellData {
				value, _ := insyra.ToFloat64Safe(v)
				allValues = append(allValues, value)
				factorsA = append(factorsA, i)
				factorsB = append(factorsB, j)
			}
		}
	}
	totalMean := insyra.NewDataList(allValues).Mean()
	totalCount := len(allValues)

	var SSA, SSB float64
	parallel.GroupUp(func() {
		for i := range factorALevels {
			var sum float64
			var count int
			for idx, a := range factorsA {
				if a == i {
					sum += allValues[idx]
					count++
				}
			}
			SSA += float64(count) * math.Pow(sum/float64(count)-totalMean, 2)
		}
	}, func() {
		for j := range factorBLevels {
			var sum float64
			var count int
			for idx, b := range factorsB {
				if b == j {
					sum += allValues[idx]
					count++
				}
			}
			SSB += float64(count) * math.Pow(sum/float64(count)-totalMean, 2)
		}
	}).Run().AwaitResult()

	cellMeans := make([]float64, len(cells))
	for i := range cells {
		cellMeans[i] = cells[i].Mean()
	}
	aMeans := make([]float64, factorALevels)
	bMeans := make([]float64, factorBLevels)
	for i := range factorALevels {
		var sum float64
		var count int
		for idx, a := range factorsA {
			if a == i {
				sum += allValues[idx]
				count++
			}
		}
		aMeans[i] = sum / float64(count)
	}
	for j := range factorBLevels {
		var sum float64
		var count int
		for idx, b := range factorsB {
			if b == j {
				sum += allValues[idx]
				count++
			}
		}
		bMeans[j] = sum / float64(count)
	}

	var SSAB, SSW float64
	for i := range factorALevels {
		for j := range factorBLevels {
			idx := i*factorBLevels + j
			exp := aMeans[i] + bMeans[j] - totalMean
			SSAB += float64(cellCounts[idx]) * (cellMeans[idx] - exp) * (cellMeans[idx] - exp)
			for _, v := range cells[idx].Data() {
				x, _ := insyra.ToFloat64Safe(v)
				SSW += (x - cellMeans[idx]) * (x - cellMeans[idx])
			}
		}
	}

	DFA := factorALevels - 1
	DFB := factorBLevels - 1
	DFAxB := DFA * DFB
	DFW := totalCount - factorALevels*factorBLevels

	FA := SSA / float64(DFA) / (SSW / float64(DFW))
	FB := SSB / float64(DFB) / (SSW / float64(DFW))
	FAB := SSAB / float64(DFAxB) / (SSW / float64(DFW))

	fd := func(d1, d2 float64, f float64) float64 {
		return 1 - distuv.F{D1: d1, D2: d2}.CDF(f)
	}

	return &TwoWayANOVAResult{
		FactorA:     ANOVAResultComponent{SSA, DFA, FA, fd(float64(DFA), float64(DFW), FA), SSA / (SSA + SSW)},
		FactorB:     ANOVAResultComponent{SSB, DFB, FB, fd(float64(DFB), float64(DFW), FB), SSB / (SSB + SSW)},
		Interaction: ANOVAResultComponent{SSAB, DFAxB, FAB, fd(float64(DFAxB), float64(DFW), FAB), SSAB / (SSAB + SSW)},
		Within:      ANOVAResultComponent{SSW, DFW, math.NaN(), math.NaN(), math.NaN()},
		TotalSS:     SSA + SSB + SSAB + SSW,
	}
}

func RepeatedMeasuresANOVA(subjects ...insyra.IDataList) *RepeatedMeasuresANOVAResult {
	if len(subjects) < 2 {
		insyra.LogWarning("stats", "RepeatedMeasuresANOVA", "At least two subjects are required")
		return nil
	}
	conditionCount := subjects[0].Len()
	for i, subj := range subjects {
		if subj.Len() != conditionCount {
			insyra.LogWarning("stats", "RepeatedMeasuresANOVA", "Inconsistent condition count at subject %d", i)
			return nil
		}
	}
	if conditionCount < 2 {
		insyra.LogWarning("stats", "RepeatedMeasuresANOVA", "Less than two conditions")
		return nil
	}

	data := make([][]float64, conditionCount)
	for i := range data {
		data[i] = make([]float64, len(subjects))
	}
	for j, subj := range subjects {
		for i, v := range subj.Data() {
			value, _ := insyra.ToFloat64Safe(v)
			data[i][j] = value
		}
	}

	var grandTotal float64
	for i := range data {
		for j := range data[i] {
			grandTotal += data[i][j]
		}
	}
	grandMean := grandTotal / float64(conditionCount*len(subjects))

	var ssTotal, ssBetween, ssSubjects float64
	parallel.GroupUp(func() {
		for i := range data {
			for j := range data[i] {
				ssTotal += (data[i][j] - grandMean) * (data[i][j] - grandMean)
			}
		}
	}, func() {
		for i := range data {
			conditionMean := 0.0
			for j := range data[i] {
				conditionMean += data[i][j]
			}
			conditionMean /= float64(len(subjects))
			ssBetween += float64(len(subjects)) * (conditionMean - grandMean) * (conditionMean - grandMean)
		}
	}, func() {
		for j := range subjects {
			subjectMean := 0.0
			for i := range data {
				subjectMean += data[i][j]
			}
			subjectMean /= float64(conditionCount)
			ssSubjects += float64(conditionCount) * (subjectMean - grandMean) * (subjectMean - grandMean)
		}
	}).Run().AwaitResult()

	SSWithin := ssTotal - ssBetween - ssSubjects

	DFBetween := conditionCount - 1
	DFSubjects := len(subjects) - 1
	DFWithin := DFBetween * DFSubjects

	MSBetween := ssBetween / float64(DFBetween)
	MSWithin := SSWithin / float64(DFWithin)

	F := MSBetween / MSWithin
	P := 1 - distuv.F{D1: float64(DFBetween), D2: float64(DFWithin)}.CDF(F)

	return &RepeatedMeasuresANOVAResult{
		Factor:  ANOVAResultComponent{ssBetween, DFBetween, F, P, ssBetween / ssTotal},
		Subject: ANOVAResultComponent{ssSubjects, DFSubjects, math.NaN(), math.NaN(), math.NaN()},
		Within:  ANOVAResultComponent{SSWithin, DFWithin, math.NaN(), math.NaN(), math.NaN()},
		TotalSS: ssTotal,
	}
}
