package nn

import (
	"fmt"
	"math"
)

// EdgeTopology is an immutable directed edge list over a fixed number of
// nodes: edge e runs from sources[e] to targets[e]. It is built once by
// NewEdgeTopology and never changes, so it can be shared between calls.
type EdgeTopology struct {
	nodes         int
	sources       []int32
	targets       []int32
	targetOffsets []int32 // len nodes+1; edges into node t are targetEdges[targetOffsets[t]:targetOffsets[t+1]]
	targetEdges   []int32 // edge indices grouped by target, ascending within each target
	sourceOffsets []int32 // same layout, grouped by source
	sourceEdges   []int32
}

// NewEdgeTopology builds an immutable edge topology over nodes nodes: edge e
// runs from sources[e] to targets[e]. It validates every index, copies the
// input slices, and precomputes the edges grouped by target and by source.
func NewEdgeTopology(nodes int, sources, targets []int) (*EdgeTopology, error) {
	if nodes <= 0 {
		return nil, fmt.Errorf("edge topology needs at least one node, got %d", nodes)
	}
	if nodes > math.MaxInt32 {
		return nil, fmt.Errorf("edge topology supports at most %d nodes, got %d", math.MaxInt32, nodes)
	}
	if len(sources) != len(targets) {
		return nil, fmt.Errorf("edge topology has %d sources and %d targets", len(sources), len(targets))
	}
	if len(sources) > math.MaxInt32 {
		return nil, fmt.Errorf("edge topology supports at most %d edges, got %d", math.MaxInt32, len(sources))
	}
	for e := 0; e < len(sources); e++ {
		if sources[e] < 0 || sources[e] >= nodes {
			return nil, fmt.Errorf("edge %d source %d is outside [0, %d)", e, sources[e], nodes)
		}
		if targets[e] < 0 || targets[e] >= nodes {
			return nil, fmt.Errorf("edge %d target %d is outside [0, %d)", e, targets[e], nodes)
		}
	}

	top := &EdgeTopology{nodes: nodes}
	edges := len(sources)
	top.sources = make([]int32, edges)
	top.targets = make([]int32, edges)
	for e := 0; e < edges; e++ {
		top.sources[e] = int32(sources[e])
		top.targets[e] = int32(targets[e])
	}

	top.targetOffsets = make([]int32, nodes+1)
	top.sourceOffsets = make([]int32, nodes+1)
	for e := 0; e < edges; e++ {
		top.targetOffsets[top.targets[e]+1]++
		top.sourceOffsets[top.sources[e]+1]++
	}
	for n := 0; n < nodes; n++ {
		top.targetOffsets[n+1] += top.targetOffsets[n]
		top.sourceOffsets[n+1] += top.sourceOffsets[n]
	}

	top.targetEdges = make([]int32, edges)
	top.sourceEdges = make([]int32, edges)
	nextTarget := make([]int32, nodes)
	nextSource := make([]int32, nodes)
	copy(nextTarget, top.targetOffsets[:nodes])
	copy(nextSource, top.sourceOffsets[:nodes])
	for e := 0; e < edges; e++ {
		t := top.targets[e]
		top.targetEdges[nextTarget[t]] = int32(e)
		nextTarget[t]++
		s := top.sources[e]
		top.sourceEdges[nextSource[s]] = int32(e)
		nextSource[s]++
	}
	return top, nil
}

// Nodes returns the number of nodes in the topology.
func (g *EdgeTopology) Nodes() int { return g.nodes }

// Edges returns the number of edges in the topology.
func (g *EdgeTopology) Edges() int { return len(g.sources) }
