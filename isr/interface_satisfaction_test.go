package isr_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
	"github.com/HazelnutParadise/insyra/stats"
)

// isr's tables and lists satisfy the core interfaces, so they go straight to
// stats, plot and Merge without unwrapping (#208).
func TestIsrTypesSatisfyTheCoreInterfaces(t *testing.T) {
	table := isr.DT.From(isr.CSV{String: "x,y\n1,2\n2,4\n3,7\n"})
	var asTable insyra.IDataTable = table
	var asList insyra.IDataList = isr.DL.From(1, 2, 3)
	_ = asList

	if _, err := stats.PCA(table); err != nil {
		t.Fatalf("stats.PCA with an isr table: %v", err)
	}
	merged, err := insyra.NewDataTable().Merge(asTable, insyra.MergeDirectionVertical, insyra.MergeModeOuter)
	if err != nil {
		t.Fatalf("Merge with an isr table: %v", err)
	}
	if rows, _ := merged.Size(); rows != 3 {
		t.Fatalf("merged %d rows, want 3", rows)
	}
	// isr keeps its own chaining ClearErr.
	table.ClearErr().Push(isr.Row{isr.Name("x"): 4, isr.Name("y"): 9})
}
