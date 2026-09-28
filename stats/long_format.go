package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// Layer 4 — long-format entry points. A long-format table has one row per
// observation; these functions group its rows and hand the groups to the
// list-taking tests, so the numbers come from one implementation.

// longFactor is one factor column of a long-format table: its distinct
// values in the order they first appear, and the level of every row as an
// index into them.
type longFactor struct {
	levels []any
	of     []int
}

// readLongFormat reads a long-format table, one row per observation: the
// finite numbers in valueCol, and one factor per entry of factorCols, named
// in errors by the matching entry of roles. The columns are resolved on a
// copy of dt, so a selector that does not resolve comes back as an error and
// nothing is recorded on dt's Err().
func readLongFormat(dt insyra.IDataTable, valueCol any, factorCols []any, roles []string) ([]float64, []longFactor, error) {
	if dt == nil {
		return nil, nil, errors.New("table is nil")
	}
	if concrete, ok := dt.(*insyra.DataTable); ok && concrete == nil {
		return nil, nil, errors.New("table is nil")
	}

	snapshot := dt.Clone()
	rows := snapshot.NumRows()

	valCol := snapshot.GetCol(valueCol)
	if valCol == nil {
		info := snapshot.PopErr()
		msg := "not found"
		if info != nil {
			msg = info.Message
		}
		return nil, nil, fmt.Errorf("value column: %s", msg)
	}

	rawVals := valCol.Data()
	cells := make([]any, rows)
	copy(cells, rawVals)

	values := make([]float64, 0, rows)
	var err error
	values, err = appendNumericValues(values, cells, func() string {
		return fmt.Sprintf("value column %v", valueCol)
	}, "row")
	if err != nil {
		return nil, nil, err
	}

	factors := make([]longFactor, len(factorCols))
	for k := range factorCols {
		facCol := snapshot.GetCol(factorCols[k])
		if facCol == nil {
			info := snapshot.PopErr()
			msg := "not found"
			if info != nil {
				msg = info.Message
			}
			return nil, nil, fmt.Errorf("%s column: %s", roles[k], msg)
		}

		facRaw := facCol.Data()
		facCells := make([]any, len(facRaw))
		copy(facCells, facRaw)

		levelMap := make(map[any]int)
		levels := make([]any, 0)
		of := make([]int, rows)

		for i := range rows {
			cell := facCells[i]
			if cell == nil {
				return nil, nil, fmt.Errorf("%s column %v has no level at row %d", roles[k], factorCols[k], i+1)
			}
			if f, ok := cell.(float64); ok && math.IsNaN(f) {
				return nil, nil, fmt.Errorf("%s column %v has no level at row %d", roles[k], factorCols[k], i+1)
			}
			if f, ok := cell.(float32); ok && math.IsNaN(float64(f)) {
				return nil, nil, fmt.Errorf("%s column %v has no level at row %d", roles[k], factorCols[k], i+1)
			}

			key := insyra.ToMapKey(cell)
			if idx, ok := levelMap[key]; ok {
				of[i] = idx
			} else {
				idx = len(levels)
				levelMap[key] = idx
				levels = append(levels, cell)
				of[i] = idx
			}
		}

		factors[k] = longFactor{levels: levels, of: of}
	}

	return values, factors, nil
}

// TwoWayANOVAFromTable runs TwoWayANOVA on a long-format table: one row per
// observation, the measured value in valueCol and the levels of the two
// factors in factorACol and factorBCol. Each column parameter is a column
// selector: an Excel-style index string ("A", "B", ...), an insyra.Name, or
// a 0-based int position. A factor's distinct values are its levels,
// compared the way insyra.ToMapKey compares them, so integers of any width
// are equal by value while floats and text are levels of their own. Every
// value has to be a finite number and every level cell has to hold a value
// other than nil or NaN; rows are counted from one in errors. Each factor
// needs at least two levels and every combination of levels at least one
// observation. The result is TwoWayANOVA's for the cells in row-major order,
// with each factor's levels in the order they first appear in the table.
// The table is not modified.
func TwoWayANOVAFromTable(dt insyra.IDataTable, valueCol, factorACol, factorBCol any) (*TwoWayANOVAResult, error) {
	values, factors, err := readLongFormat(dt, valueCol, []any{factorACol, factorBCol}, []string{"factor A", "factor B"})
	if err != nil {
		return nil, err
	}

	a := factors[0]
	b := factors[1]

	if len(a.levels) < 2 {
		return nil, errors.New("factor A has fewer than two levels")
	}
	if len(b.levels) < 2 {
		return nil, errors.New("factor B has fewer than two levels")
	}

	nCells := len(a.levels) * len(b.levels)
	cellValues := make([][]float64, nCells)

	for r, v := range values {
		idx := a.of[r]*len(b.levels) + b.of[r]
		cellValues[idx] = append(cellValues[idx], v)
	}

	cells := make([]insyra.IDataList, nCells)
	for i := range a.levels {
		for j := range b.levels {
			idx := i*len(b.levels) + j
			if len(cellValues[idx]) == 0 {
				return nil, fmt.Errorf("no observations for A=%v, B=%v", a.levels[i], b.levels[j])
			}
			cells[idx] = insyra.NewDataList(cellValues[idx])
		}
	}

	return TwoWayANOVA(len(a.levels), len(b.levels), cells)
}

// readRepeatedMeasures reads a long-format repeated-measures table into one
// list per subject holding that subject's value under each condition, the
// subjects and the conditions in the order they first appear. Every subject
// needs exactly one observation under every condition.
func readRepeatedMeasures(dt insyra.IDataTable, valueCol, conditionCol, subjectCol any) ([]insyra.IDataList, error) {
	values, factors, err := readLongFormat(dt, valueCol, []any{conditionCol, subjectCol}, []string{"condition", "subject"})
	if err != nil {
		return nil, err
	}
	cond, subj := factors[0], factors[1]
	k, n := len(cond.levels), len(subj.levels)
	if k < 2 {
		return nil, errors.New("at least two conditions are required")
	}
	if n < 2 {
		return nil, errors.New("at least two subjects are required")
	}
	rowOf := make([][]int, n)
	for i := range rowOf {
		rowOf[i] = make([]int, k)
		for j := range rowOf[i] {
			rowOf[i][j] = -1
		}
	}
	grid := make([][]float64, n)
	for i := range grid {
		grid[i] = make([]float64, k)
	}
	for r, v := range values {
		s := subj.of[r]
		c := cond.of[r]
		if rowOf[s][c] >= 0 {
			return nil, fmt.Errorf("subject %v has more than one observation for condition %v (rows %d and %d)", subj.levels[s], cond.levels[c], rowOf[s][c]+1, r+1)
		}
		rowOf[s][c] = r
		grid[s][c] = v
	}
	for s := 0; s < n; s++ {
		for c := 0; c < k; c++ {
			if rowOf[s][c] < 0 {
				return nil, fmt.Errorf("subject %v has no observation for condition %v", subj.levels[s], cond.levels[c])
			}
		}
	}
	lists := make([]insyra.IDataList, n)
	for s := 0; s < n; s++ {
		lists[s] = insyra.NewDataList(grid[s])
	}
	return lists, nil
}

// RepeatedMeasuresANOVAFromTable runs RepeatedMeasuresANOVA on a
// long-format table: one row per observation, the measured value in
// valueCol, the condition in conditionCol and the subject in subjectCol.
// Each column parameter is a column selector: an Excel-style index string,
// an insyra.Name, or a 0-based int position. Levels are compared the way
// insyra.ToMapKey compares them. Every value has to be a finite number,
// every condition and subject cell a value other than nil or NaN, and every
// subject needs exactly one observation under every condition; rows are
// counted from one in errors. The result is RepeatedMeasuresANOVA's for the
// subjects and conditions in the order they first appear. The table is not
// modified.
func RepeatedMeasuresANOVAFromTable(dt insyra.IDataTable, valueCol, conditionCol, subjectCol any) (*RepeatedMeasuresANOVAResult, error) {
	lists, err := readRepeatedMeasures(dt, valueCol, conditionCol, subjectCol)
	if err != nil {
		return nil, err
	}
	return RepeatedMeasuresANOVA(lists)
}

// FriedmanTestFromTable runs FriedmanTest on a long-format table, read the
// way RepeatedMeasuresANOVAFromTable reads it.
func FriedmanTestFromTable(dt insyra.IDataTable, valueCol, conditionCol, subjectCol any) (*FriedmanTestResult, error) {
	lists, err := readRepeatedMeasures(dt, valueCol, conditionCol, subjectCol)
	if err != nil {
		return nil, err
	}
	return FriedmanTest(lists)
}
