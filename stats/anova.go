package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parallel"
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

func newANOVAComponent(sumOfSquares float64, df int, f, p, eta float64) ANOVAResultComponent {
	return ANOVAResultComponent{
		SumOfSquares: sumOfSquares,
		DF:           df,
		F:            f,
		P:            p,
		EtaSquared:   eta,
	}
}

func newANOVAWithinComponent(sumOfSquares float64, df int) ANOVAResultComponent {
	return newANOVAComponent(sumOfSquares, df, math.NaN(), math.NaN(), math.NaN())
}

func newANOVABetweenComponent(ssEffect float64, dfEffect int, ssWithin float64, dfWithin int) ANOVAResultComponent {
	f := fRatio(ssEffect, dfEffect, ssWithin, dfWithin)
	p := fOneTailedPValue(f, float64(dfEffect), float64(dfWithin))
	eta := etaSquared(ssEffect, ssWithin)
	return newANOVAComponent(ssEffect, dfEffect, f, p, eta)
}

func OneWayANOVA(groups ...insyra.IDataList) *OneWayANOVAResult {
	if len(groups) < 2 {
		insyra.LogWarning("stats", "OneWayANOVA", "At least two groups are required")
		return nil
	}

	values := make([]float64, 0)
	labels := make([]int, 0)
	for i, g := range groups {
		var groupData []any
		g.AtomicDo(func(gdl *insyra.DataList) {
			groupData = gdl.Data()
		})

		if len(groupData) == 0 {
			insyra.LogWarning("stats", "OneWayANOVA", "Group %d is empty", i)
			return nil
		}
		for j, v := range groupData {
			x, ok := insyra.ToFloat64Safe(v)
			if !ok {
				insyra.LogWarning("stats", "OneWayANOVA", "Invalid data at group %d index %d", i, j)
				return nil
			}
			values = append(values, x)
			labels = append(labels, i)
		}
	}

	stats, err := oneWayANOVAFromSlices(values, labels, len(groups))
	if err != nil {
		insyra.LogWarning("stats", "OneWayANOVA", "%s", err)
		return nil
	}

	return &OneWayANOVAResult{
		Factor:  newANOVAComponent(stats.SSB, stats.DFB, stats.F, stats.P, stats.Eta),
		Within:  newANOVAWithinComponent(stats.SSW, stats.DFW),
		TotalSS: stats.SSB + stats.SSW,
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

	factorA := newANOVABetweenComponent(SSA, DFA, SSW, DFW)
	factorB := newANOVABetweenComponent(SSB, DFB, SSW, DFW)
	interaction := newANOVABetweenComponent(SSAB, DFAxB, SSW, DFW)

	return &TwoWayANOVAResult{
		FactorA:     factorA,
		FactorB:     factorB,
		Interaction: interaction,
		Within:      newANOVAWithinComponent(SSW, DFW),
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

	F := fRatio(ssBetween, DFBetween, SSWithin, DFWithin)
	P := fOneTailedPValue(F, float64(DFBetween), float64(DFWithin))

	return &RepeatedMeasuresANOVAResult{
		Factor:  newANOVAComponent(ssBetween, DFBetween, F, P, ssBetween/ssTotal),
		Subject: newANOVAWithinComponent(ssSubjects, DFSubjects),
		Within:  newANOVAWithinComponent(SSWithin, DFWithin),
		TotalSS: ssTotal,
	}
}
