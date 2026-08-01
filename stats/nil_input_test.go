package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// TestClusteringEntryPointsRejectNilTables pins a guard that was missing until
// 2026-07-29: every clustering and decomposition entry point loads its input
// through one shared helper, and a nil table reached that helper's AtomicDo and
// panicked instead of returning an error. A panic is not a usable error for a
// library caller, and the entry points are numerous enough that guarding each
// one separately would drift.
func TestClusteringEntryPointsRejectNilTables(t *testing.T) {
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	var typedNil *insyra.DataTable

	fitted, err := stats.KMeans(insyra.NewDataTable(
		insyra.NewDataList(1.0, 2.0, 8.0, 9.0),
		insyra.NewDataList(1.0, 2.0, 8.0, 9.0),
	), 2)
	if err != nil {
		t.Fatalf("kmeans: %v", err)
	}

	cases := []struct {
		name string
		call func(insyra.IDataTable) error
	}{
		{"KMeans", func(dt insyra.IDataTable) error { _, err := stats.KMeans(dt, 2); return err }},
		{"DBSCAN", func(dt insyra.IDataTable) error { _, err := stats.DBSCAN(dt, 1, 2); return err }},
		{"PCA", func(dt insyra.IDataTable) error { _, err := stats.PCA(dt, 1); return err }},
		{"HierarchicalAgglomerative", func(dt insyra.IDataTable) error {
			_, err := stats.HierarchicalAgglomerative(dt, stats.AggloComplete)
			return err
		}},
		{"Silhouette", func(dt insyra.IDataTable) error {
			_, err := stats.Silhouette(dt, insyra.NewDataList(1, 1, 2, 2))
			return err
		}},
		{"KMeansResult.Assign", func(dt insyra.IDataTable) error { _, _, err := fitted.Assign(dt); return err }},
	}

	for _, c := range cases {
		for _, input := range []struct {
			kind  string
			table insyra.IDataTable
		}{{"nil interface", nil}, {"typed nil", typedNil}} {
			t.Run(c.name+"/"+input.kind, func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panicked instead of returning an error: %v", r)
					}
				}()
				if err := c.call(input.table); err == nil {
					t.Fatal("expected an error for a nil table")
				}
			})
		}
	}
}
