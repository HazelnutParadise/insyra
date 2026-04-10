package stats_test

import (
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestKMeansRejectsInvalidCenters(t *testing.T) {
	dt := dataTableFromRows([][]float64{
		{0, 0},
		{1, 1},
	})

	if _, err := stats.KMeans(dt, 0); err == nil {
		t.Fatalf("expected error for centers <= 0")
	}
	if _, err := stats.KMeans(dt, 3); err == nil {
		t.Fatalf("expected error for centers > n")
	}
}

func TestKMeansReturnsClusterSummary(t *testing.T) {
	dt := dataTableFromRows([][]float64{
		{0, 0},
		{0, 1},
		{1, 0},
		{10, 10},
		{10, 11},
		{11, 10},
	})
	seed := int64(7)

	got, err := stats.KMeans(dt, 2, stats.KMeansOptions{NStart: 3, IterMax: 20, Seed: &seed})
	if err != nil {
		t.Fatalf("KMeans returned error: %v", err)
	}

	if len(got.Cluster) != 6 {
		t.Fatalf("expected 6 cluster assignments, got %d", len(got.Cluster))
	}
	if len(got.Size) != 2 {
		t.Fatalf("expected 2 cluster sizes, got %d", len(got.Size))
	}
	if len(got.WithinSS) != 2 {
		t.Fatalf("expected 2 within-cluster ss values, got %d", len(got.WithinSS))
	}
	if got.Centers == nil {
		t.Fatalf("expected centers table")
	}
	r, c := got.Centers.(*insyra.DataTable).Size()
	if r != 2 || c != 2 {
		t.Fatalf("expected centers shape 2x2, got %dx%d", r, c)
	}
	if got.TotSS <= 0 || got.TotWithinSS <= 0 || got.BetweenSS <= 0 {
		t.Fatalf("expected positive ss summary, got tot=%v within=%v between=%v", got.TotSS, got.TotWithinSS, got.BetweenSS)
	}
	if got.Iter < 1 {
		t.Fatalf("expected positive iteration count, got %d", got.Iter)
	}
}

func TestHierarchicalAgglomerativeAndCutTree(t *testing.T) {
	dt := dataTableFromRows([][]float64{
		{0, 0},
		{0, 1},
		{10, 10},
		{10, 11},
	})

	tree, err := stats.HierarchicalAgglomerative(dt, stats.AggloComplete)
	if err != nil {
		t.Fatalf("HierarchicalAgglomerative returned error: %v", err)
	}
	if len(tree.Merge) != 3 {
		t.Fatalf("expected 3 merge rows, got %d", len(tree.Merge))
	}
	if len(tree.Height) != 3 || len(tree.Order) != 4 || len(tree.Labels) != 4 {
		t.Fatalf("unexpected hierarchical result sizes: heights=%d order=%d labels=%d", len(tree.Height), len(tree.Order), len(tree.Labels))
	}
	if tree.Method != stats.AggloComplete {
		t.Fatalf("unexpected method: %s", tree.Method)
	}
	if tree.DistMethod != "euclidean" {
		t.Fatalf("unexpected distance method: %s", tree.DistMethod)
	}

	byK, err := stats.CutTreeByK(tree, 2)
	if err != nil {
		t.Fatalf("CutTreeByK returned error: %v", err)
	}
	if len(byK) != 4 {
		t.Fatalf("expected 4 cut labels, got %d", len(byK))
	}
	if byK[0] != byK[1] || byK[2] != byK[3] || byK[0] == byK[2] {
		t.Fatalf("unexpected k-cut labels: %v", byK)
	}

	byHeight, err := stats.CutTreeByHeight(tree, 5)
	if err != nil {
		t.Fatalf("CutTreeByHeight returned error: %v", err)
	}
	if !reflect.DeepEqual(byK, byHeight) {
		t.Fatalf("expected same cut for k and height, got %v vs %v", byK, byHeight)
	}
}

func TestDBSCANDetectsNoiseAndCorePoints(t *testing.T) {
	dt := dataTableFromRows([][]float64{
		{0, 0},
		{0.1, 0},
		{0, 0.1},
		{8, 8},
	})

	got, err := stats.DBSCAN(dt, 0.25, 3)
	if err != nil {
		t.Fatalf("DBSCAN returned error: %v", err)
	}
	if !reflect.DeepEqual(got.Cluster, []int{1, 1, 1, 0}) {
		t.Fatalf("unexpected cluster labels: %v", got.Cluster)
	}
	if !reflect.DeepEqual(got.IsSeed, []bool{true, true, true, false}) {
		t.Fatalf("unexpected seed labels: %v", got.IsSeed)
	}
}

func TestSilhouetteReturnsPerPointDetails(t *testing.T) {
	dt := dataTableFromRows([][]float64{
		{0, 0},
		{0, 1},
		{10, 10},
		{10, 11},
	})
	labels := insyra.NewDataList(1, 1, 2, 2)

	got, err := stats.Silhouette(dt, labels)
	if err != nil {
		t.Fatalf("Silhouette returned error: %v", err)
	}
	if len(got.Points) != 4 {
		t.Fatalf("expected 4 silhouette points, got %d", len(got.Points))
	}
	for i, pt := range got.Points {
		if pt.Cluster != labels.Get(i).(int) {
			t.Fatalf("point %d cluster mismatch: %+v", i, pt)
		}
		if pt.Neighbor == pt.Cluster {
			t.Fatalf("point %d expected different neighbor cluster: %+v", i, pt)
		}
	}
	if got.AverageSilhouette <= 0 {
		t.Fatalf("expected positive average silhouette, got %v", got.AverageSilhouette)
	}
}

func TestSilhouetteRejectsSingleCluster(t *testing.T) {
	dt := dataTableFromRows([][]float64{
		{0, 0},
		{0, 1},
		{1, 0},
	})
	labels := insyra.NewDataList(1, 1, 1)

	if _, err := stats.Silhouette(dt, labels); err == nil {
		t.Fatalf("expected error for single-cluster silhouette")
	}
}
