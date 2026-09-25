package nn

import (
	"math"
	"math/rand"
	"runtime"
	"slices"
	"testing"
)

func TestNewEdgeTopologyRefuses(t *testing.T) {
	tests := []struct {
		name    string
		nodes   int
		sources []int
		targets []int
		wantErr string
	}{
		{
			name:    "non-positive node count",
			nodes:   0,
			wantErr: "edge topology needs at least one node, got 0",
		},
		{
			name:    "mismatched source and target count",
			nodes:   2,
			sources: []int{0},
			targets: []int{1, 0},
			wantErr: "edge topology has 1 sources and 2 targets",
		},
		{
			name:    "source out of range",
			nodes:   2,
			sources: []int{2},
			targets: []int{0},
			wantErr: "edge 0 source 2 is outside [0, 2)",
		},
		{
			name:    "target out of range",
			nodes:   2,
			sources: []int{0},
			targets: []int{2},
			wantErr: "edge 0 target 2 is outside [0, 2)",
		},
		{
			name:    "negative source index",
			nodes:   2,
			sources: []int{-1},
			targets: []int{0},
			wantErr: "edge 0 source -1 is outside [0, 2)",
		},
		{
			name:    "second edge target out of range",
			nodes:   3,
			sources: []int{0, 1},
			targets: []int{1, 3},
			wantErr: "edge 1 target 3 is outside [0, 3)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEdgeTopology(tt.nodes, tt.sources, tt.targets)
			if err == nil {
				t.Fatalf("NewEdgeTopology(%d, %v, %v) succeeded, want error %q", tt.nodes, tt.sources, tt.targets, tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("NewEdgeTopology(%d, %v, %v) error = %q, want %q", tt.nodes, tt.sources, tt.targets, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewEdgeTopologyGroupsEdges(t *testing.T) {
	top, err := NewEdgeTopology(4, []int{2, 0, 2, 1, 0}, []int{1, 1, 3, 1, 0})
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	if got := top.Nodes(); got != 4 {
		t.Fatalf("Nodes() = %d, want 4", got)
	}
	if got := top.Edges(); got != 5 {
		t.Fatalf("Edges() = %d, want 5", got)
	}
	if want := []int32{0, 1, 4, 4, 5}; !slices.Equal(top.targetOffsets, want) {
		t.Fatalf("targetOffsets = %v, want %v", top.targetOffsets, want)
	}
	if want := []int32{4, 0, 1, 3, 2}; !slices.Equal(top.targetEdges, want) {
		t.Fatalf("targetEdges = %v, want %v", top.targetEdges, want)
	}
	if want := []int32{0, 2, 3, 5, 5}; !slices.Equal(top.sourceOffsets, want) {
		t.Fatalf("sourceOffsets = %v, want %v", top.sourceOffsets, want)
	}
	if want := []int32{1, 4, 3, 0, 2}; !slices.Equal(top.sourceEdges, want) {
		t.Fatalf("sourceEdges = %v, want %v", top.sourceEdges, want)
	}
}

func TestNewEdgeTopologyAllowsNoEdges(t *testing.T) {
	top, err := NewEdgeTopology(3, nil, nil)
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	if got := top.Nodes(); got != 3 {
		t.Fatalf("Nodes() = %d, want 3", got)
	}
	if got := top.Edges(); got != 0 {
		t.Fatalf("Edges() = %d, want 0", got)
	}
	want := []int32{0, 0, 0, 0}
	if !slices.Equal(top.targetOffsets, want) {
		t.Fatalf("targetOffsets = %v, want %v", top.targetOffsets, want)
	}
	if !slices.Equal(top.sourceOffsets, want) {
		t.Fatalf("sourceOffsets = %v, want %v", top.sourceOffsets, want)
	}
}

func TestNewEdgeTopologyCopiesItsInput(t *testing.T) {
	sources := []int{2, 0, 2, 1, 0}
	targets := []int{1, 1, 3, 1, 0}
	top, err := NewEdgeTopology(4, sources, targets)
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	sources[0] = 999
	targets[0] = 999
	if got := top.sources[0]; got != 2 {
		t.Fatalf("topology sources[0] after caller mutation = %d, want 2", got)
	}
	if got := top.targets[0]; got != 1 {
		t.Fatalf("topology targets[0] after caller mutation = %d, want 1", got)
	}
}

func TestEdgeSumIssueExample(t *testing.T) {
	top, err := NewEdgeTopology(3, []int{0, 1, 2}, []int{1, 2, 0})
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	weights, err := NewFloat32Tensor([]int{3}, []float32{0.5, -1, 0.25})
	if err != nil {
		t.Fatalf("NewFloat32Tensor(weights): %v", err)
	}
	values, err := NewFloat32Tensor([]int{3}, []float32{0.1, 0.2, 0.3})
	if err != nil {
		t.Fatalf("NewFloat32Tensor(values): %v", err)
	}
	out, err := EdgeSum(top, weights, values)
	if err != nil {
		t.Fatalf("EdgeSum: %v", err)
	}
	if !slices.Equal(out.Shape(), []int{3}) {
		t.Fatalf("output shape = %v, want [3]", out.Shape())
	}
	want := []float32{
		float32(float32(0.25) * float32(0.3)),
		float32(float32(0.5) * float32(0.1)),
		float32(float32(-1) * float32(0.2)),
	}
	if !slices.Equal(out.Data(), want) {
		t.Fatalf("output = %v, want %v", out.Data(), want)
	}
}

func TestEdgeSumMatchesDenseReference(t *testing.T) {
	sources := []int{0, 1, 2, 3, 4, 5, 0, 1, 2, 0, 5, 5, 3, 1, 4}
	targets := []int{1, 1, 1, 2, 2, 0, 0, 3, 3, 3, 3, 4, 4, 5, 5}
	top, err := NewEdgeTopology(6, sources, targets)
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	weights32 := []float32{0.7, -1.2, 2.3, 0.4, -3.1, 1.1, -0.6, 2.2, -1.7, 0.9, 1.6, -0.8, 2.5, -1.4, 0.3}
	values32 := []float32{0.15, -0.25, 0.33, -0.72, 0.9, -1.1}
	weights, err := NewFloat32Tensor([]int{len(sources)}, weights32)
	if err != nil {
		t.Fatalf("NewFloat32Tensor(weights): %v", err)
	}
	values, err := NewFloat32Tensor([]int{6}, values32)
	if err != nil {
		t.Fatalf("NewFloat32Tensor(values): %v", err)
	}
	out, err := EdgeSum(top, weights, values)
	if err != nil {
		t.Fatalf("EdgeSum: %v", err)
	}

	var dense [6][6]float64
	for e := range sources {
		dense[targets[e]][sources[e]] += float64(weights32[e])
	}
	ref := make([]float64, 6)
	for t := 0; t < 6; t++ {
		for s := 0; s < 6; s++ {
			ref[t] += dense[t][s] * float64(values32[s])
		}
	}
	got := out.Data()
	for i := 0; i < 6; i++ {
		tolerance := 1e-6 * (1 + math.Abs(ref[i]))
		if math.Abs(float64(got[i])-ref[i]) > tolerance {
			t.Fatalf("output[%d] = %v, dense float64 reference %v out of tolerance %v", i, got[i], ref[i], tolerance)
		}
	}
}

func TestEdgeSumMatchesExactOracle(t *testing.T) {
	sources := []int{0, 1, 2, 3, 4, 5, 0, 1, 2, 0, 5, 5, 3, 1, 4}
	targets := []int{1, 1, 1, 2, 2, 0, 0, 3, 3, 3, 3, 4, 4, 5, 5}
	fixedTopology, err := NewEdgeTopology(6, sources, targets)
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	fixedWeights := []float32{0.7, -1.2, 2.3, 0.4, -3.1, 1.1, -0.6, 2.2, -1.7, 0.9, 1.6, -0.8, 2.5, -1.4, 0.3}
	fixedValues := []float32{
		0.15, -0.25, 0.33, -0.72, 0.9, -1.1,
		-0.4, 0.55, 0.77, -0.12, 0.31, 0.64,
		2.1, 1.5, -0.9, 0.47, -0.23, 3.3,
	}
	checkFixed := func() {
		t.Helper()
		weights := mustTestTensor(t, []int{len(fixedWeights)}, fixedWeights)
		values := mustTestTensor(t, []int{3, 6}, fixedValues)
		out, err := EdgeSum(fixedTopology, weights, values)
		if err != nil {
			t.Fatalf("fixed graph: EdgeSum: %v", err)
		}
		if !slices.Equal(out.Shape(), []int{3, 6}) {
			t.Fatalf("fixed graph: output shape = %v, want [3 6]", out.Shape())
		}
		for b := 0; b < 3; b++ {
			for target := 0; target < 6; target++ {
				var pairs [][2]float32
				for e := range sources {
					if targets[e] == target {
						pairs = append(pairs, [2]float32{fixedWeights[e], fixedValues[b*6+sources[e]]})
					}
				}
				got := math.Float32bits(out.Data()[b*6+target])
				want := math.Float32bits(exactSumOracle(pairs))
				if got != want {
					t.Fatalf("fixed graph output[%d][%d]: bits = %#08x, want %#08x", b, target, got, want)
				}
			}
		}
	}
	checkFixed()

	const (
		nodes       = 500
		edges       = 20_000
		targetNodes = 50
		batch       = 2
	)
	r := rand.New(rand.NewSource(3))
	randomSources := make([]int, edges)
	randomTargets := make([]int, edges)
	for e := range edges {
		randomSources[e] = r.Intn(nodes)
		randomTargets[e] = r.Intn(targetNodes)
	}
	randomWeights := make([]float32, edges)
	for i := range randomWeights {
		randomWeights[i] = float32(math.Ldexp(r.NormFloat64(), r.Intn(41)-20))
	}
	randomValues := make([]float32, batch*nodes)
	for i := range randomValues {
		randomValues[i] = float32(math.Ldexp(r.NormFloat64(), r.Intn(41)-20))
	}
	randomTopology, err := NewEdgeTopology(nodes, randomSources, randomTargets)
	if err != nil {
		t.Fatalf("random graph: NewEdgeTopology: %v", err)
	}
	randomWeightTensor := mustTestTensor(t, []int{edges}, randomWeights)
	randomValueTensor := mustTestTensor(t, []int{batch, nodes}, randomValues)
	randomOut, err := EdgeSum(randomTopology, randomWeightTensor, randomValueTensor)
	if err != nil {
		t.Fatalf("random graph: EdgeSum: %v", err)
	}
	for b := 0; b < batch; b++ {
		for target := 0; target < nodes; target++ {
			var pairs [][2]float32
			for e := range edges {
				if randomTargets[e] == target {
					pairs = append(pairs, [2]float32{randomWeights[e], randomValues[b*nodes+randomSources[e]]})
				}
			}
			got := math.Float32bits(randomOut.Data()[b*nodes+target])
			want := math.Float32bits(exactSumOracle(pairs))
			if got != want {
				t.Fatalf("random graph output[%d][%d]: bits = %#08x, want %#08x", b, target, got, want)
			}
		}
	}
}

func TestEdgeSumNodeWithoutIncomingEdges(t *testing.T) {
	top, err := NewEdgeTopology(3, []int{0}, []int{1})
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	weights, err := NewFloat32Tensor([]int{1}, []float32{2})
	if err != nil {
		t.Fatalf("NewFloat32Tensor(weights): %v", err)
	}
	values, err := NewFloat32Tensor([]int{3}, []float32{1, 2, 3})
	if err != nil {
		t.Fatalf("NewFloat32Tensor(values): %v", err)
	}
	out, err := EdgeSum(top, weights, values)
	if err != nil {
		t.Fatalf("EdgeSum: %v", err)
	}
	if want := []float32{0, 2, 0}; !slices.Equal(out.Data(), want) {
		t.Fatalf("output = %v, want %v", out.Data(), want)
	}
}

func TestEdgeSumRefuses(t *testing.T) {
	top, err := NewEdgeTopology(3, []int{0, 1, 2}, []int{1, 2, 0})
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	goodWeights, err := NewFloat32Tensor([]int{3}, []float32{0.5, -1, 0.25})
	if err != nil {
		t.Fatalf("NewFloat32Tensor(weights): %v", err)
	}
	goodValues, err := NewFloat32Tensor([]int{3}, []float32{0.1, 0.2, 0.3})
	if err != nil {
		t.Fatalf("NewFloat32Tensor(values): %v", err)
	}
	boolValues := mustTestBoolTensor(t, []int{3}, []bool{true, false, true})

	tests := []struct {
		name    string
		top     *EdgeTopology
		weights *Tensor
		values  *Tensor
		wantErr string
	}{
		{
			name:    "nil topology",
			wantErr: "edge sum topology is nil",
		},
		{
			name:    "weights length wrong",
			top:     top,
			weights: mustTestTensor(t, []int{2}, []float32{0.5, -1}),
			values:  goodValues,
			wantErr: "edge sum weights must have shape [3], got [2]",
		},
		{
			name:    "weights rank 2",
			top:     top,
			weights: mustTestTensor(t, []int{1, 3}, []float32{0.5, -1, 0.25}),
			values:  goodValues,
			wantErr: "edge sum weights must have shape [3], got [1 3]",
		},
		{
			name:    "values length not N",
			top:     top,
			weights: goodWeights,
			values:  mustTestTensor(t, []int{4}, []float32{0.1, 0.2, 0.3, 0.4}),
			wantErr: "edge sum values must have shape [3] or [B, 3], got [4]",
		},
		{
			name:    "values rank 3",
			top:     top,
			weights: goodWeights,
			values:  mustTestTensor(t, []int{1, 1, 3}, []float32{0.1, 0.2, 0.3}),
			wantErr: "edge sum values must have shape [3] or [B, 3], got [1 1 3]",
		},
		{
			name:    "values bool tensor",
			top:     top,
			weights: goodWeights,
			values:  boolValues,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EdgeSum(tt.top, tt.weights, tt.values)
			if err == nil {
				t.Fatalf("EdgeSum succeeded, want error %q", tt.wantErr)
			}
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Fatalf("EdgeSum error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestEdgeSumLargeSparseGraph(t *testing.T) {
	nodes := 1_000_000
	top, err := NewEdgeTopology(nodes, []int{0, 999_999, 5, 5}, []int{999_999, 0, 7, 7})
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	weights, err := NewFloat32Tensor([]int{4}, []float32{1, 2, 3, 4})
	if err != nil {
		t.Fatalf("NewFloat32Tensor(weights): %v", err)
	}
	valuesData := make([]float32, nodes)
	for i := range valuesData {
		valuesData[i] = float32(i % 10)
	}
	values, err := NewFloat32Tensor([]int{nodes}, valuesData)
	if err != nil {
		t.Fatalf("NewFloat32Tensor(values): %v", err)
	}

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	out, err := EdgeSum(top, weights, values)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("EdgeSum: %v", err)
	}
	if alloc := int64(after.TotalAlloc - before.TotalAlloc); alloc >= 64<<20 {
		t.Fatalf("EdgeSum allocated %d bytes, want less than %d", alloc, 64<<20)
	}

	if !slices.Equal(out.Shape(), []int{nodes}) {
		t.Fatalf("output shape = %v, want [%d]", out.Shape(), nodes)
	}
	got := out.Data()
	if got[999_999] != float32(float32(1)*valuesData[0]) {
		t.Fatalf("output[999999] = %v, want %v", got[999_999], float32(float32(1)*valuesData[0]))
	}
	if got[0] != float32(float32(2)*valuesData[999_999]) {
		t.Fatalf("output[0] = %v, want %v", got[0], float32(float32(2)*valuesData[999_999]))
	}
	want7 := float32(float32(3)*valuesData[5]) + float32(float32(4)*valuesData[5])
	if got[7] != want7 {
		t.Fatalf("output[7] = %v, want %v", got[7], want7)
	}
	for _, i := range []int{1, 3, 6, 8, 123_456, 888_888, 999_998} {
		if got[i] != 0 {
			t.Fatalf("output[%d] = %v, want 0 (no incoming edges)", i, got[i])
		}
	}
}
