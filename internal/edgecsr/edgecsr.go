// Package edgecsr groups the edges of a directed edge list by target and by
// source, for nn's EdgeTopology and accel's DeviceEdgeTopology.
package edgecsr

// Grouped is an edge list grouped both ways. Edges into node t are
// TargetEdges[TargetOffsets[t]:TargetOffsets[t+1]], ascending; edges leaving
// node s are SourceEdges[SourceOffsets[s]:SourceOffsets[s+1]], ascending.
type Grouped struct {
	TargetOffsets, TargetEdges []int32
	SourceOffsets, SourceEdges []int32
}

// Group builds both groupings. Every source and target must already lie in
// [0, nodes); callers validate them.
func Group(nodes int, sources, targets []int32) Grouped {
	g := Grouped{
		TargetOffsets: make([]int32, nodes+1),
		SourceOffsets: make([]int32, nodes+1),
		TargetEdges:   make([]int32, len(sources)),
		SourceEdges:   make([]int32, len(sources)),
	}
	for e := range sources {
		g.TargetOffsets[targets[e]+1]++
		g.SourceOffsets[sources[e]+1]++
	}
	for n := 0; n < nodes; n++ {
		g.TargetOffsets[n+1] += g.TargetOffsets[n]
		g.SourceOffsets[n+1] += g.SourceOffsets[n]
	}
	nextTarget := make([]int32, nodes)
	nextSource := make([]int32, nodes)
	copy(nextTarget, g.TargetOffsets[:nodes])
	copy(nextSource, g.SourceOffsets[:nodes])
	for e := range sources {
		t := targets[e]
		g.TargetEdges[nextTarget[t]] = int32(e)
		nextTarget[t]++
		s := sources[e]
		g.SourceEdges[nextSource[s]] = int32(e)
		nextSource[s]++
	}
	return g
}
