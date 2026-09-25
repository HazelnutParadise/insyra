package insyra_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// A caller may extend the core table by embedding it; the result still goes
// wherever an IDataTable goes, Merge included (#208, #228).
type salesTable struct {
	*insyra.DataTable
	Region string
}

func TestAnEmbeddingTypeGoesWhereATableGoes(t *testing.T) {
	sales := salesTable{
		DataTable: insyra.NewDataTable(
			insyra.NewDataList(1.0, 2.0, 3.0).SetName("x"),
			insyra.NewDataList(2.0, 4.0, 7.0).SetName("y"),
		),
		Region: "north",
	}
	var _ insyra.IDataTable = sales

	other := insyra.NewDataTable(insyra.NewDataList(4.0).SetName("x"), insyra.NewDataList(8.0).SetName("y"))
	merged, err := other.Merge(sales, insyra.MergeDirectionVertical, insyra.MergeModeOuter)
	if err != nil {
		t.Fatalf("Merge with an embedding type: %v", err)
	}
	if rows, _ := merged.Size(); rows != 4 {
		t.Fatalf("merged %d rows, want 4", rows)
	}
	if _, err := other.Merge(sales, insyra.MergeDirectionHorizontal, insyra.MergeModeInner, "x"); err != nil {
		t.Fatalf("horizontal Merge with an embedding type: %v", err)
	}
	if _, err := stats.PCA(sales); err != nil {
		t.Fatalf("stats.PCA with an embedding type: %v", err)
	}
}
