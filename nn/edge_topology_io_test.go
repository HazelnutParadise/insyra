package nn

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
)

func edgeTopologyFixture(t *testing.T) *EdgeTopology {
	t.Helper()
	g, err := NewEdgeTopology(5, []int{0, 4, 2, 2, 1, 3}, []int{1, 1, 0, 4, 3, 3})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func savedEdgeTopology(t *testing.T, g *EdgeTopology) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := SaveEdgeTopology(&buffer, g); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestEdgeTopologyRoundTrip(t *testing.T) {
	g := edgeTopologyFixture(t)
	data := savedEdgeTopology(t, g)
	loaded, err := LoadEdgeTopology(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Equal(g) || !g.Equal(loaded) {
		t.Fatal("a loaded topology is not Equal to the saved one")
	}
	if loaded.Device().ID != "" {
		t.Fatalf("a loaded topology lives on %q, want the host", loaded.Device().ID)
	}
	if again := savedEdgeTopology(t, loaded); !bytes.Equal(again, data) {
		t.Fatal("saving a loaded topology gives different bytes")
	}
	// A topology without edges round-trips too.
	empty, err := NewEdgeTopology(3, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if loaded, err := LoadEdgeTopology(bytes.NewReader(savedEdgeTopology(t, empty))); err != nil || !loaded.Equal(empty) {
		t.Fatalf("an edgeless topology: %v", err)
	}
}

func TestEdgeTopologyEqualComparesEdgeOrder(t *testing.T) {
	g := edgeTopologyFixture(t)
	same, _ := NewEdgeTopology(5, []int{0, 4, 2, 2, 1, 3}, []int{1, 1, 0, 4, 3, 3})
	swapped, _ := NewEdgeTopology(5, []int{4, 0, 2, 2, 1, 3}, []int{1, 1, 0, 4, 3, 3}) // same edges, two swapped
	moreNodes, _ := NewEdgeTopology(6, []int{0, 4, 2, 2, 1, 3}, []int{1, 1, 0, 4, 3, 3})
	fewer, _ := NewEdgeTopology(5, []int{0, 4, 2, 2, 1}, []int{1, 1, 0, 4, 3})
	var none *EdgeTopology
	for name, c := range map[string]struct {
		other *EdgeTopology
		want  bool
	}{
		"same": {same, true}, "swapped": {swapped, false}, "more nodes": {moreNodes, false},
		"fewer edges": {fewer, false}, "nil": {none, false},
	} {
		if got := g.Equal(c.other); got != c.want {
			t.Errorf("%s: Equal = %v, want %v", name, got, c.want)
		}
	}
	if !none.Equal(nil) {
		t.Error("two nil topologies are not Equal")
	}
}

// rewriteEdgeTopology decodes a saved file, lets edit change its header and
// data, and encodes it again.
func rewriteEdgeTopology(t *testing.T, data []byte, edit func(header map[string]any, payload []byte) []byte) []byte {
	t.Helper()
	n := binary.LittleEndian.Uint64(data[:8])
	var header map[string]any
	if err := json.Unmarshal(data[8:8+n], &header); err != nil {
		t.Fatal(err)
	}
	payload := edit(header, append([]byte(nil), data[8+n:]...))
	encoded, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]byte, 8, 8+len(encoded)+len(payload))
	binary.LittleEndian.PutUint64(out, uint64(len(encoded)))
	return append(append(out, encoded...), payload...)
}

func TestLoadEdgeTopologyRefusesABadFile(t *testing.T) {
	data := savedEdgeTopology(t, edgeTopologyFixture(t))
	offsets := func(header map[string]any, name string) []any {
		return header[name].(map[string]any)["data_offsets"].([]any)
	}
	putInt := func(payload []byte, header map[string]any, name string, index int, value int64) []byte {
		begin := int(offsets(header, name)[0].(float64))
		binary.LittleEndian.PutUint64(payload[begin+8*index:], uint64(value))
		return payload
	}
	cases := map[string]struct {
		file []byte
		want string
	}{
		"empty":     {nil, "read header length"},
		"truncated": {data[:len(data)-3], "data is"},
		"trailing":  {append(append([]byte(nil), data...), 0), "trailing data"},
		"optimizer file": {rewriteEdgeTopology(t, data, func(h map[string]any, p []byte) []byte {
			h["__metadata__"] = map[string]any{"insyra.optimizer_state.version": "2"}
			return p
		}), "not an edge-topology file"},
		"newer version": {rewriteEdgeTopology(t, data, func(h map[string]any, p []byte) []byte {
			h["__metadata__"] = map[string]any{edgeTopologyVersionKey: "2"}
			return p
		}), "unsupported version"},
		"unknown tensor": {rewriteEdgeTopology(t, data, func(h map[string]any, p []byte) []byte {
			h["weights"] = h["nodes"]
			return p
		}), "unknown tensor"},
		"missing tensor": {rewriteEdgeTopology(t, data, func(h map[string]any, p []byte) []byte {
			delete(h, "targets")
			return p
		}), "missing tensor"},
		"wrong dtype": {rewriteEdgeTopology(t, data, func(h map[string]any, p []byte) []byte {
			h["nodes"].(map[string]any)["dtype"] = "F64"
			return p
		}), "want I64"},
		"source out of range": {rewriteEdgeTopology(t, data, func(h map[string]any, p []byte) []byte {
			return putInt(p, h, "sources", 2, 5)
		}), "sources[2] = 5 is outside"},
		"negative target": {rewriteEdgeTopology(t, data, func(h map[string]any, p []byte) []byte {
			return putInt(p, h, "targets", 0, -1)
		}), "targets[0] = -1 is outside"},
		"no nodes": {rewriteEdgeTopology(t, data, func(h map[string]any, p []byte) []byte {
			return putInt(p, h, "nodes", 0, 0)
		}), "node count 0"},
		// A header that claims far more data than the file holds is refused
		// after reading what is there, not by allocating what it claims.
		"claims a huge payload": {rewriteEdgeTopology(t, data, func(h map[string]any, p []byte) []byte {
			const edges = 1 << 30
			h["sources"] = map[string]any{"dtype": "I64", "shape": []int{edges}, "data_offsets": []int64{8, 8 + 8*edges}}
			h["targets"] = map[string]any{"dtype": "I64", "shape": []int{edges}, "data_offsets": []int64{8 + 8*edges, 8 + 16*edges}}
			h["nodes"] = map[string]any{"dtype": "I64", "shape": []int{}, "data_offsets": []int64{0, 8}}
			return p
		}), "data is"},
	}
	for name, c := range cases {
		g, err := LoadEdgeTopology(bytes.NewReader(c.file))
		if err == nil || g != nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, %v; want an error containing %q", name, g, err, c.want)
		}
	}
	if _, err := LoadEdgeTopology(nil); err == nil {
		t.Error("a nil reader was accepted")
	}
	if err := SaveEdgeTopology(&bytes.Buffer{}, nil); err == nil {
		t.Error("a nil topology was saved")
	}
	if err := SaveEdgeTopology(nil, edgeTopologyFixture(t)); err == nil {
		t.Error("a nil writer was accepted")
	}
}
