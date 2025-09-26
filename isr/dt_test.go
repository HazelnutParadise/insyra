package isr_test

import (
	"testing"

	. "github.com/HazelnutParadise/insyra/isr"
	"github.com/stretchr/testify/assert"
)

func TestDT_Push(t *testing.T) {
	dt := DT.From(
		DLs{
			DL.From(1, 2, 3).SetName("A"),
			DL.From("x", "y", "z").SetName("B"),
		},
	)
	dt.Push(
		DL.From(4, 5, 6).SetName("C"),
	).Push(
		Rows{
			{"A": 1, "B": "a", "C": 10.5},
			{"A": 2, "B": "b", "C": 20.5},
		},
	).Push(
		Cols{
			{0: 7, 1: 8, 2: 9},
			{0: "m", 1: "n", 2: "o"},
		},
	)
	numRow, numCol := dt.Size()
	assert.Equal(t, 5, len(dt.ColNames()), "Expected 5 columns after pushing")
	assert.Equal(t, 5, numCol, "Expected 5 columns after pushing")
	assert.Equal(t, 5, numRow, "Expected 5 rows after pushing")
	assert.Equal(t, []string{"A", "B", "C", "", ""}, dt.ColNames(), "Column names do not match")
	assert.Equal(t, []any{1, 2, 3, 1, 2}, dt.GetColByName("A").Data(), "Column A data does not match")
	assert.Equal(t, []any{"x", "y", "z", "a", "b"}, dt.GetColByName("B").Data(), "Column B data does not match")
	assert.Equal(t, []any{4, 5, 6, 10.5, 20.5}, dt.GetColByName("C").Data(), "Column C data does not match")
	assert.Equal(t, []any{7, 8, 9, nil, nil}, dt.GetCol("D").Data(), "Column D data does not match")
	assert.Equal(t, []any{"m", "n", "o", nil, nil}, dt.GetCol("E").Data(), "Column E data does not match")
}
