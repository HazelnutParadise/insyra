package nn

import "testing"

// A nil topology, such as the one NewEdgeTopology returns with an error, answers
// its accessors instead of panicking.
func TestNilEdgeTopologyAccessors(t *testing.T) {
	var top *EdgeTopology
	if n := top.Nodes(); n != 0 {
		t.Errorf("Nodes() = %d, want 0", n)
	}
	if n := top.Edges(); n != 0 {
		t.Errorf("Edges() = %d, want 0", n)
	}
}
