package stats

import (
	"errors"
	"fmt"
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

func OneWayANOVA(groups ...insyra.IDataList) (*OneWayANOVAResult, error) {
	if len(groups) < 2 {
		return nil, errors.New("at least two groups are required")
	}

	values := make([]float64, 0)
	labels := make([]int, 0)
	for i, g := range groups {
		var groupData []any
		g.AtomicDo(func(gdl *insyra.DataList) {
			groupData = gdl.Data()
		})

		if len(groupData) == 0 {
			return nil, fmt.Errorf("group %d is empty", i)
		}
		for j, v := range groupData {
			x, ok := insyra.ToFloat64Safe(v)
			if !ok {
				return nil, fmt.Errorf("invalid data at group %d index %d", i, j)
			}
			values = append(values, x)
			labels = append(labels, i)
		}
	}

	stats, err := oneWayANOVAFromSlices(values, labels, len(groups))
	if err != nil {
		return nil, err
	}

	return &OneWayANOVAResult{
		Factor:  newANOVAComponent(stats.SSB, stats.DFB, stats.F, stats.P, stats.Eta),
		Within:  newANOVAWithinComponent(stats.SSW, stats.DFW),
		TotalSS: stats.SSB + stats.SSW,
	}, nil
}

func TwoWayANOVA(factorALevels, factorBLevels int, cells ...insyra.IDataList) (*TwoWayANOVAResult, error) {
	if factorALevels < 2 || factorBLevels < 2 || len(cells) != factorALevels*factorBLevels {
		return nil, errors.New("invalid levels or cells")
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
				return nil, fmt.Errorf("empty cell at A=%d, B=%d", i, j)
			}
			cellCounts[i*factorBLevels+j] = cellLen
			for k, v := range cellData {
				value, ok := insyra.ToFloat64Safe(v)
				if !ok {
					return nil, fmt.Errorf("invalid data at cell (A=%d, B=%d) index %d", i, j, k)
				}
				allValues = append(allValues, value)
				factorsA = append(factorsA, i)
				factorsB = append(factorsB, j)
			}
		}
	}
	totalMean := insyra.NewDataList(allValues).Mean()
	totalCount := len(allValues)

	// Pre-aggregate sums and counts per A-row and per B-column in a single
	// pass over the values. Eliminates the previous O(N × levels) inner-loop
	// pattern (and its math.Pow(_, 2) calls — see stats/CLAUDE.md "禁止 inline
	// math.Pow with integer exponent 2").
	sumsA := make([]float64, factorALevels)
	sumsB := make([]float64, factorBLevels)
	countsA := make([]int, factorALevels)
	countsB := make([]int, factorBLevels)
	for idx, v := range allValues {
		sumsA[factorsA[idx]] += v
		sumsB[factorsB[idx]] += v
		countsA[factorsA[idx]]++
		countsB[factorsB[idx]]++
	}
	aMeans := make([]float64, factorALevels)
	bMeans := make([]float64, factorBLevels)
	var SSA, SSB float64
	for i := range factorALevels {
		aMeans[i] = sumsA[i] / float64(countsA[i])
		dev := aMeans[i] - totalMean
		SSA += float64(countsA[i]) * dev * dev
	}
	for j := range factorBLevels {
		bMeans[j] = sumsB[j] / float64(countsB[j])
		dev := bMeans[j] - totalMean
		SSB += float64(countsB[j]) * dev * dev
	}

	cellMeans := make([]float64, len(cells))
	for i := range cells {
		cellMeans[i] = cells[i].Mean()
	}

	var SSAB, SSW float64
	for i := range factorALevels {
		for j := range factorBLevels {
			idx := i*factorBLevels + j
			exp := aMeans[i] + bMeans[j] - totalMean
			SSAB += float64(cellCounts[idx]) * (cellMeans[idx] - exp) * (cellMeans[idx] - exp)
			for k, v := range cells[idx].Data() {
				x, ok := insyra.ToFloat64Safe(v)
				if !ok {
					return nil, fmt.Errorf("invalid data at cell %d index %d", idx, k)
				}
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
	}, nil
}

func RepeatedMeasuresANOVA(subjects ...insyra.IDataList) (*RepeatedMeasuresANOVAResult, error) {
	if len(subjects) < 2 {
		return nil, errors.New("at least two subjects are required")
	}
	conditionCount := subjects[0].Len()
	for i, subj := range subjects {
		if subj.Len() != conditionCount {
			return nil, fmt.Errorf("inconsistent condition count at subject %d", i)
		}
	}
	if conditionCount < 2 {
		return nil, errors.New("less than two conditions")
	}

	data := make([][]float64, conditionCount)
	for i := range data {
		data[i] = make([]float64, len(subjects))
	}
	for j, subj := range subjects {
		for i, v := range subj.Data() {
			value, ok := insyra.ToFloat64Safe(v)
			if !ok {
				return nil, fmt.Errorf("invalid data at subject %d condition %d", j, i)
			}
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
	}, nil
}
