package stats_test

import (
	"math"
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

// ============================================================
// R reference suite — Batch 9
// ============================================================
//
// Tolerances: hclust merge/height/order uses Euclidean distances which depend
// on the same FP order as R; with distinct distance values (test data is
// chosen to avoid distance ties) the results match R to ~1e-13.

const (
	tolHclust = 1e-12
	tolSil    = 1e-12
)

var clusterRef = &refTable{path: "testdata/clustering_reference.txt"}
var clusterDump = &labelledFloats{path: "testdata/clustering_data_dump.txt"}

func TestHierarchicalAgglomerative_R(t *testing.T) {
	smallData := [][]float64{
		{0.0, 0.0}, {0.5, 1.1}, {8.3, 7.9}, {9.2, 8.4}, {15.1, 0.7},
	}
	cases := []struct {
		name   string
		method stats.AgglomerativeMethod
		prefix string
		data   [][]float64
	}{
		{"complete_n5", stats.AggloComplete, "hc_complete", smallData},
		{"single_n5", stats.AggloSingle, "hc_single", smallData},
		{"average_n5", stats.AggloAverage, "hc_average", smallData},
		{"wardD_n5", stats.AggloWardD, "hc_ward.D", smallData},
		{"wardD2_n5", stats.AggloWardD2, "hc_ward.D2", smallData},
		{"complete_n10_3D", stats.AggloComplete, "hc10_complete", largeHclustData(t)},
		{"average_n10_3D", stats.AggloAverage, "hc10_average", largeHclustData(t)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dt := dataTableFromRows(c.data)
			tree, err := stats.HierarchicalAgglomerative(dt, c.method)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			n := len(c.data)
			if len(tree.Merge) != n-1 {
				t.Fatalf("merge length: got %d, want %d", len(tree.Merge), n-1)
			}
			for i := range n - 1 {
				expA := int(clusterRef.get(t, c.prefix+".merge["+itoa(i)+"].a"))
				expB := int(clusterRef.get(t, c.prefix+".merge["+itoa(i)+"].b"))
				if tree.Merge[i][0] != expA || tree.Merge[i][1] != expB {
					t.Errorf("merge[%d]: got [%d %d], want [%d %d]",
						i, tree.Merge[i][0], tree.Merge[i][1], expA, expB)
				}
				expH := clusterRef.get(t, c.prefix+".height["+itoa(i)+"]")
				if !pClose(tree.Height[i], expH, tolHclust) {
					t.Errorf("height[%d]: got %.17g, want %.17g (Δ=%g)",
						i, tree.Height[i], expH, math.Abs(tree.Height[i]-expH))
				}
			}
			for i := range n {
				expO := int(clusterRef.get(t, c.prefix+".order["+itoa(i)+"]"))
				if tree.Order[i] != expO {
					t.Errorf("order[%d]: got %d, want %d", i, tree.Order[i], expO)
				}
			}
		})
	}
}

func largeHclustData(t *testing.T) [][]float64 {
	t.Helper()
	rows := make([][]float64, 10)
	for i := range rows {
		rows[i] = clusterDump.get(t, "hc10_row"+itoa(i))
	}
	return rows
}

func TestSilhouette_R(t *testing.T) {
	cases := []struct {
		name   string
		data   [][]float64
		labels []int
		prefix string
	}{
		{"clean_two_clusters",
			[][]float64{
				{0, 0}, {0.2, 0.1}, {0.1, 0.3},
				{10, 10}, {10.1, 9.9}, {9.9, 10.1},
			},
			[]int{1, 1, 1, 2, 2, 2},
			"sil_clean"},
		{"mixed_with_singleton",
			[][]float64{
				{0, 0}, {0.5, 0}, {0, 0.5},
				{5, 5}, {5.3, 4.8}, {4.7, 5.2}, {5.1, 5.4},
				{20, 20},
			},
			[]int{1, 1, 1, 2, 2, 2, 2, 3},
			"sil_mixed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dt := dataTableFromRows(c.data)
			lblData := make([]any, len(c.labels))
			for i, v := range c.labels {
				lblData[i] = v
			}
			labels := insyra.NewDataList(lblData...)
			r, err := stats.Silhouette(dt, labels)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if len(r.Points) != len(c.data) {
				t.Fatalf("points length: got %d, want %d", len(r.Points), len(c.data))
			}
			for i, pt := range r.Points {
				expS := clusterRef.get(t, c.prefix+".s["+itoa(i)+"]")
				if !pClose(pt.SilWidth, expS, tolSil) {
					t.Errorf("s[%d]: got %.17g, want %.17g", i, pt.SilWidth, expS)
				}
				if pt.Cluster != c.labels[i] {
					t.Errorf("cluster[%d]: got %d, want %d", i, pt.Cluster, c.labels[i])
				}
			}
			expAvg := clusterRef.get(t, c.prefix+".avg")
			if !pClose(r.AverageSilhouette, expAvg, tolSil) {
				t.Errorf("avg: got %.17g, want %.17g", r.AverageSilhouette, expAvg)
			}
		})
	}
}
