package nn

import (
	"math"
	"math/rand"
	"strings"
	"testing"
)

func TestTapeEdgeSumMatchesExactOracle(t *testing.T) {
	sources := []int{0, 1, 2, 3, 4, 5, 0, 1, 2, 0, 5, 5, 3, 1, 4}
	targets := []int{1, 1, 1, 2, 2, 0, 0, 3, 3, 3, 3, 4, 4, 5, 5}
	top, err := NewEdgeTopology(6, sources, targets)
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	weightsF := []float32{0.7, -1.2, 2.3, 0.4, -3.1, 1.1, -0.6, 2.2, -1.7, 0.9, 1.6, -0.8, 2.5, -1.4, 0.3}
	valuesF := []float32{
		0.15, -0.25, 0.33, -0.72, 0.9, -1.1,
		-0.4, 0.55, 0.77, -0.12, 0.31, 0.64,
		2.1, 1.5, -0.9, 0.47, -0.23, 3.3,
	}
	upstreamF := []float32{
		0.3, -0.4, 0.8, 0.5, -0.7, 0.2,
		0.1, 0.6, -0.9, 0.4, 0.2, -0.1,
		0.7, -0.3, 0.9, 0.5, -0.2, 0.8,
	}
	w := mustTestTensor(t, []int{15}, weightsF)
	v := mustTestTensor(t, []int{3, 6}, valuesF)
	g := mustTestTensor(t, []int{3, 6}, upstreamF)
	tape := NewTape()
	if _, err := tape.Param(w); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(v); err != nil {
		t.Fatal(err)
	}
	y, err := tape.EdgeSum(top, w, v)
	if err != nil {
		t.Fatalf("tape.EdgeSum: %v", err)
	}
	if err := tape.BackwardFrom(y, g); err != nil {
		t.Fatalf("BackwardFrom: %v", err)
	}

	gradV, err := tape.Grad(v)
	if err != nil {
		t.Fatal(err)
	}
	gradW, err := tape.Grad(w)
	if err != nil {
		t.Fatal(err)
	}
	for b := 0; b < 3; b++ {
		for s := 0; s < 6; s++ {
			var pairs [][2]float32
			for e := range sources {
				if sources[e] == s {
					pairs = append(pairs, [2]float32{weightsF[e], upstreamF[b*6+targets[e]]})
				}
			}
			got := math.Float32bits(gradV.Data()[b*6+s])
			want := math.Float32bits(exactSumOracle(pairs))
			if got != want {
				t.Fatalf("values gradient[%d][%d]: bits = %#08x, want %#08x", b, s, got, want)
			}
		}
	}
	for e := range weightsF {
		var pairs [][2]float32
		for b := 0; b < 3; b++ {
			pairs = append(pairs, [2]float32{upstreamF[b*6+targets[e]], valuesF[b*6+sources[e]]})
		}
		got := math.Float32bits(gradW.Data()[e])
		want := math.Float32bits(exactSumOracle(pairs))
		if got != want {
			t.Fatalf("weights gradient[%d]: bits = %#08x, want %#08x", e, got, want)
		}
	}
}

func TestEdgeSumEdgeOrderDoesNotMatter(t *testing.T) {
	sources := []int{0, 1, 2, 3, 4, 5, 0, 1, 2, 0, 5, 5, 3, 1, 4}
	targets := []int{1, 1, 1, 2, 2, 0, 0, 3, 3, 3, 3, 4, 4, 5, 5}
	weightsF := []float32{0.7, -1.2, 2.3, 0.4, -3.1, 1.1, -0.6, 2.2, -1.7, 0.9, 1.6, -0.8, 2.5, -1.4, 0.3}
	valuesF := []float32{
		0.15, -0.25, 0.33, -0.72, 0.9, -1.1,
		-0.4, 0.55, 0.77, -0.12, 0.31, 0.64,
		2.1, 1.5, -0.9, 0.47, -0.23, 3.3,
	}
	upstreamF := []float32{
		0.3, -0.4, 0.8, 0.5, -0.7, 0.2,
		0.1, 0.6, -0.9, 0.4, 0.2, -0.1,
		0.7, -0.3, 0.9, 0.5, -0.2, 0.8,
	}
	order := rand.New(rand.NewSource(17)).Perm(len(sources))
	permutedSources := make([]int, len(sources))
	permutedTargets := make([]int, len(targets))
	permutedWeights := make([]float32, len(weightsF))
	for edge, original := range order {
		permutedSources[edge] = sources[original]
		permutedTargets[edge] = targets[original]
		permutedWeights[edge] = weightsF[original]
	}
	originalTopology, err := NewEdgeTopology(6, sources, targets)
	if err != nil {
		t.Fatalf("original NewEdgeTopology: %v", err)
	}
	permutedTopology, err := NewEdgeTopology(6, permutedSources, permutedTargets)
	if err != nil {
		t.Fatalf("permuted NewEdgeTopology: %v", err)
	}

	runForward := func(topology *EdgeTopology, weightsData []float32) *Tensor {
		t.Helper()
		weights := mustTestTensor(t, []int{len(weightsData)}, weightsData)
		values := mustTestTensor(t, []int{3, 6}, valuesF)
		out, err := EdgeSum(topology, weights, values)
		if err != nil {
			t.Fatalf("EdgeSum: %v", err)
		}
		return out
	}
	originalForward := runForward(originalTopology, weightsF)
	permutedForward := runForward(permutedTopology, permutedWeights)
	for i := range originalForward.Data() {
		got := math.Float32bits(permutedForward.Data()[i])
		want := math.Float32bits(originalForward.Data()[i])
		if got != want {
			t.Fatalf("permuted forward[%d]: bits = %#08x, want %#08x", i, got, want)
		}
	}

	type tapeGradients struct {
		dValues  *Tensor
		dWeights *Tensor
	}
	runTape := func(topology *EdgeTopology, weightsData []float32) tapeGradients {
		t.Helper()
		weights := mustTestTensor(t, []int{len(weightsData)}, weightsData)
		values := mustTestTensor(t, []int{3, 6}, valuesF)
		upstream := mustTestTensor(t, []int{3, 6}, upstreamF)
		tape := NewTape()
		if _, err := tape.Param(weights); err != nil {
			t.Fatal(err)
		}
		if _, err := tape.Param(values); err != nil {
			t.Fatal(err)
		}
		y, err := tape.EdgeSum(topology, weights, values)
		if err != nil {
			t.Fatalf("tape.EdgeSum: %v", err)
		}
		if err := tape.BackwardFrom(y, upstream); err != nil {
			t.Fatalf("BackwardFrom: %v", err)
		}
		dValues, err := tape.Grad(values)
		if err != nil {
			t.Fatal(err)
		}
		dWeights, err := tape.Grad(weights)
		if err != nil {
			t.Fatal(err)
		}
		return tapeGradients{dValues: dValues, dWeights: dWeights}
	}
	originalGradients := runTape(originalTopology, weightsF)
	permutedGradients := runTape(permutedTopology, permutedWeights)
	for i := range originalGradients.dValues.Data() {
		got := math.Float32bits(permutedGradients.dValues.Data()[i])
		want := math.Float32bits(originalGradients.dValues.Data()[i])
		if got != want {
			t.Fatalf("permuted values gradient[%d]: bits = %#08x, want %#08x", i, got, want)
		}
	}
	for edge, original := range order {
		got := math.Float32bits(permutedGradients.dWeights.Data()[edge])
		want := math.Float32bits(originalGradients.dWeights.Data()[original])
		if got != want {
			t.Fatalf("permuted weights gradient[%d] maps to original edge %d: bits = %#08x, want %#08x", edge, original, got, want)
		}
	}
}

func TestTapeEdgeSumFiniteDifferences(t *testing.T) {
	sources := []int{0, 1, 2, 3, 0, 2}
	targets := []int{1, 1, 2, 0, 3, 3}
	top, err := NewEdgeTopology(4, sources, targets)
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	weightsF := []float32{0.5, -1.3, 0.9, 0.2, -0.7, 1.1}
	valuesF := []float32{0.3, -0.8, 1.2, -0.4}
	targetF := []float32{0.1, 0.6, -0.2, 0.9}
	w := mustTestTensor(t, []int{6}, weightsF)
	v := mustTestTensor(t, []int{4}, valuesF)
	target := mustTestTensor(t, []int{4}, targetF)
	tape := NewTape()
	if _, err := tape.Param(w); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(v); err != nil {
		t.Fatal(err)
	}
	edge, err := tape.EdgeSum(top, w, v)
	if err != nil {
		t.Fatalf("tape.EdgeSum: %v", err)
	}
	tanh, err := tape.Tanh(edge)
	if err != nil {
		t.Fatalf("tape.Tanh: %v", err)
	}
	loss, err := tape.MSELoss(tanh, target)
	if err != nil {
		t.Fatalf("tape.MSELoss: %v", err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatalf("Backward: %v", err)
	}
	gradW, err := tape.Grad(w)
	if err != nil {
		t.Fatal(err)
	}
	gradV, err := tape.Grad(v)
	if err != nil {
		t.Fatal(err)
	}

	lossFor := func(weightsData, valuesData []float32) float32 {
		weights := mustTestTensor(t, []int{6}, weightsData)
		values := mustTestTensor(t, []int{4}, valuesData)
		edgeResult, err := EdgeSum(top, weights, values)
		if err != nil {
			t.Fatalf("EdgeSum: %v", err)
		}
		tanhResult, err := Tanh(edgeResult)
		if err != nil {
			t.Fatalf("Tanh: %v", err)
		}
		lossResult, err := mseLossForward(tanhResult, target)
		if err != nil {
			t.Fatalf("mseLossForward: %v", err)
		}
		return lossResult.Data()[0]
	}

	const eps = float32(1e-2)
	checkWeights := func() {
		for index := range weightsF {
			plus := append([]float32(nil), weightsF...)
			minus := append([]float32(nil), weightsF...)
			plus[index] += eps
			minus[index] -= eps
			finite := (lossFor(plus, valuesF) - lossFor(minus, valuesF)) / (2 * eps)
			analytic := gradW.Data()[index]
			if math.Abs(float64(finite-analytic)) > 2e-2*(1+math.Abs(float64(analytic))) {
				t.Fatalf("weights gradient[%d] = %g, finite difference = %g", index, analytic, finite)
			}
		}
	}
	checkValues := func() {
		for index := range valuesF {
			plus := append([]float32(nil), valuesF...)
			minus := append([]float32(nil), valuesF...)
			plus[index] += eps
			minus[index] -= eps
			finite := (lossFor(weightsF, plus) - lossFor(weightsF, minus)) / (2 * eps)
			analytic := gradV.Data()[index]
			if math.Abs(float64(finite-analytic)) > 2e-2*(1+math.Abs(float64(analytic))) {
				t.Fatalf("values gradient[%d] = %g, finite difference = %g", index, analytic, finite)
			}
		}
	}
	checkWeights()
	checkValues()
}

func TestTapeEdgeSumRecordsNothingOnError(t *testing.T) {
	w := mustTestTensor(t, []int{1}, []float32{0.5})
	v := mustTestTensor(t, []int{2}, []float32{0.1, 0.2})
	tape := NewTape()
	before := len(tape.ops)
	if _, err := tape.EdgeSum(nil, w, v); err == nil {
		t.Fatalf("tape.EdgeSum with nil topology succeeded, want error")
	}
	if got := len(tape.ops); got != before {
		t.Fatalf("ops after failed EdgeSum = %d, want %d", got, before)
	}
}

func TestTapeEdgeSumRejectsMisshapedUpstream(t *testing.T) {
	top, err := NewEdgeTopology(6, []int{0}, []int{1})
	if err != nil {
		t.Fatalf("NewEdgeTopology: %v", err)
	}
	weights := mustTestTensor(t, []int{1}, []float32{0.5})
	values := mustTestTensor(t, []int{6}, []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6})
	upstream := mustTestTensor(t, []int{5}, []float32{1, 2, 3, 4, 5})
	_, err = edgeSumVJP(top, weights, values, upstream)
	if err == nil {
		t.Fatalf("edgeSumVJP with upstream [5] and values [6] succeeded, want error")
	}
	if !strings.Contains(err.Error(), "[5]") || !strings.Contains(err.Error(), "[6]") {
		t.Fatalf("error = %q, want it to name shapes [5] and [6]", err.Error())
	}
}
