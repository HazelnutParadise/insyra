package nn

import (
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
