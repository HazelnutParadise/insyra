package mkt

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
)

func TestBasketAnalysis(t *testing.T) {
	// 5 orders:
	//   O1: A, B, C
	//   O2: A, B
	//   O3: A, C
	//   O4: B, C
	//   O5: A
	// itemCounts: A=4, B=3, C=3
	// pairCounts: A∩B=2 (O1,O2), A∩C=2 (O1,O3), B∩C=2 (O1,O4)
	rows := []struct {
		order, product string
	}{
		{"O1", "A"}, {"O1", "B"}, {"O1", "C"},
		{"O2", "A"}, {"O2", "B"},
		{"O3", "A"}, {"O3", "C"},
		{"O4", "B"}, {"O4", "C"},
		{"O5", "A"},
	}

	dt := insyra.NewDataTable()
	for _, r := range rows {
		dt.AppendRowsByColIndex(map[string]any{
			"A": r.order,
			"B": r.product,
		})
	}
	dt.SetColNameByIndex("A", "OrderID")
	dt.SetColNameByIndex("B", "ProductID")

	result := BasketAnalysis(dt, BasketConfig{
		OrderIDColName:   "OrderID",
		ProductIDColName: "ProductID",
	})
	if result == nil {
		t.Fatal("BasketAnalysis returned nil")
	}

	for name, table := range map[string]insyra.IDataTable{
		"Support":    result.Support,
		"Confidence": result.Confidence,
		"Lift":       result.Lift,
	} {
		nr, nc := table.Size()
		if nr != 3 || nc != 3 {
			t.Errorf("%s matrix expected 3x3, got %dx%d", name, nr, nc)
		}
		if got := table.ColNames(); !equalStringSlice(got, []string{"A", "B", "C"}) {
			t.Errorf("%s matrix col names = %v, want [A B C]", name, got)
		}
		if got := table.RowNames(); !equalStringSlice(got, []string{"A", "B", "C"}) {
			t.Errorf("%s matrix row names = %v, want [A B C]", name, got)
		}
	}

	// Spot-check the (A, B) cell across all three matrices.
	// Support(A,B) = 2/5 = 0.4
	// Confidence(A→B) = 2/4 = 0.5
	// Lift(A→B) = 0.4 / ((4/5) * (3/5)) = 0.4 / 0.48 ≈ 0.8333...
	checks := []struct {
		matrix insyra.IDataTable
		name   string
		a, b   string
		want   float64
	}{
		{result.Support, "Support", "A", "B", 0.4},
		{result.Confidence, "Confidence", "A", "B", 0.5},
		{result.Lift, "Lift", "A", "B", 0.4 / (0.8 * 0.6)},

		// Diagonal sanity: Support(A,A) = P(A) = 4/5; Confidence(A→A) = 1
		{result.Support, "Support", "A", "A", 0.8},
		{result.Confidence, "Confidence", "A", "A", 1.0},

		// Confidence(B→A) = 2/3, asymmetric vs Confidence(A→B) = 0.5
		{result.Confidence, "Confidence", "B", "A", 2.0 / 3.0},
	}
	for _, c := range checks {
		got := getCell(t, c.matrix, c.a, c.b)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s(%s,%s) = %v, want %v", c.name, c.a, c.b, got, c.want)
		}
	}
}

func TestBasketAnalysisMissingConfig(t *testing.T) {
	dt := insyra.NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{"A": "O1", "B": "X"})

	if BasketAnalysis(dt, BasketConfig{ProductIDColIndex: "B"}) != nil {
		t.Error("expected nil when OrderID column is not provided")
	}
	if BasketAnalysis(dt, BasketConfig{OrderIDColIndex: "A"}) != nil {
		t.Error("expected nil when ProductID column is not provided")
	}
}

func getCell(t *testing.T, table insyra.IDataTable, rowName, colName string) float64 {
	t.Helper()
	rowIdx, ok := table.GetRowIndexByName(rowName)
	if !ok {
		t.Fatalf("row %q not found", rowName)
	}
	colIdx := table.GetColIndexByName(colName)
	if colIdx == "" {
		t.Fatalf("col %q not found", colName)
	}
	return conv.ParseF64(table.GetElement(rowIdx, colIdx))
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
