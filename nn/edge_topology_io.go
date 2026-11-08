package nn

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	edgeTopologyVersionKey = "insyra.edge_topology.version"
	edgeTopologyVersion    = "1"
	// An edge-topology header names three tensors and one metadata entry, so
	// a few hundred bytes are enough; the limit only stops a bad length.
	edgeTopologyMaxHeader = 4 << 10
)

// SaveEdgeTopology writes g as a SafeTensors file: the I64 tensors sources and
// targets in edge order, the I64 scalar nodes, and the format version in the
// metadata. A topology moved to a device keeps its edge list on the host, so
// it writes the same bytes as its original and nothing is read back. An I/O
// error can leave partial output.
func SaveEdgeTopology(w io.Writer, g *EdgeTopology) error {
	if nilOptimizerStateEndpoint(w) {
		return fmt.Errorf("save edge topology: writer is nil")
	}
	if g == nil {
		return fmt.Errorf("save edge topology: topology is nil")
	}
	sources := make([]int64, len(g.sources))
	targets := make([]int64, len(g.targets))
	for e := range g.sources {
		sources[e], targets[e] = int64(g.sources[e]), int64(g.targets[e])
	}
	edges := []int{len(sources)}
	sourceTensor, err := newInt64Tensor(edges, sources)
	if err != nil {
		return fmt.Errorf("save edge topology: %w", err)
	}
	targetTensor, err := newInt64Tensor(edges, targets)
	if err != nil {
		return fmt.Errorf("save edge topology: %w", err)
	}
	nodes, err := newInt64Tensor(nil, []int64{int64(g.nodes)})
	if err != nil {
		return fmt.Errorf("save edge topology: %w", err)
	}
	tensors := map[string]*Tensor{"nodes": nodes, "sources": sourceTensor, "targets": targetTensor}
	if err := saveSafeTensors(w, tensors, map[string]string{edgeTopologyVersionKey: edgeTopologyVersion}); err != nil {
		return fmt.Errorf("save edge topology: %w", err)
	}
	return nil
}

// LoadEdgeTopology reads a file SaveEdgeTopology wrote. It checks the format
// version and the three tensors' dtypes and shapes before reading their data,
// reads no more data than the file holds, and rebuilds the topology with
// NewEdgeTopology, which checks every index. The topology it returns is on the
// host; move it with To.
func LoadEdgeTopology(r io.Reader) (*EdgeTopology, error) {
	if nilOptimizerStateEndpoint(r) {
		return nil, fmt.Errorf("load edge topology: reader is nil")
	}
	var lengthBytes [8]byte
	if _, err := io.ReadFull(r, lengthBytes[:]); err != nil {
		return nil, fmt.Errorf("load edge topology: read header length: %w", err)
	}
	headerLength := binary.LittleEndian.Uint64(lengthBytes[:])
	if headerLength > edgeTopologyMaxHeader {
		return nil, fmt.Errorf("load edge topology: header length %d exceeds limit %d", headerLength, edgeTopologyMaxHeader)
	}
	header := make([]byte, int(headerLength))
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("load edge topology: read header: %w", err)
	}
	entries, metadata, err := parseSafeTensorHeader(header)
	if err != nil {
		return nil, fmt.Errorf("load edge topology: invalid SafeTensors header: %w", err)
	}
	// The version comes first, so a file from a newer release is reported as
	// such rather than by a tensor this release does not know.
	version, ok := metadata[edgeTopologyVersionKey]
	if !ok {
		return nil, fmt.Errorf("load edge topology: not an edge-topology file (no %s in its metadata)", edgeTopologyVersionKey)
	}
	if version != edgeTopologyVersion {
		return nil, fmt.Errorf("load edge topology: unsupported version %q, this release reads version %s", version, edgeTopologyVersion)
	}

	byName := make(map[string]safeTensorEntry, len(entries))
	for _, entry := range entries {
		switch entry.name {
		case "nodes", "sources", "targets":
		default:
			return nil, fmt.Errorf("load edge topology: unknown tensor %q", entry.name)
		}
		if entry.dtype != "I64" {
			return nil, fmt.Errorf("load edge topology: tensor %q has wire dtype %s, want I64", entry.name, entry.dtype)
		}
		byName[entry.name] = entry
	}
	for _, name := range []string{"nodes", "sources", "targets"} {
		if _, ok := byName[name]; !ok {
			return nil, fmt.Errorf("load edge topology: missing tensor %q", name)
		}
	}
	if shape := byName["nodes"].shape; len(shape) != 0 {
		return nil, fmt.Errorf("load edge topology: tensor \"nodes\" has shape %v, want a scalar", shape)
	}
	sourceShape, targetShape := byName["sources"].shape, byName["targets"].shape
	if len(sourceShape) != 1 || len(targetShape) != 1 || sourceShape[0] != targetShape[0] {
		return nil, fmt.Errorf("load edge topology: sources has shape %v and targets %v, want the same [edges]", sourceShape, targetShape)
	}
	if sourceShape[0] > math.MaxInt32 {
		return nil, fmt.Errorf("load edge topology: %d edges exceed the supported %d", sourceShape[0], math.MaxInt32)
	}

	regions := make([]safeTensorRegion, 0, len(entries))
	maxEnd := uint64(0)
	for _, entry := range entries {
		if entry.begin > entry.end || entry.end-entry.begin != entry.count*8 {
			return nil, fmt.Errorf("load edge topology: tensor %q has data_offsets [%d, %d] for %d values", entry.name, entry.begin, entry.end, entry.count)
		}
		regions = append(regions, safeTensorRegion{name: entry.name, begin: entry.begin, end: entry.end})
		maxEnd = max(maxEnd, entry.end)
	}
	if err := validateSafeTensorRegions(regions, maxEnd); err != nil {
		return nil, fmt.Errorf("load edge topology: invalid data regions: %w", err)
	}
	// Read through a limit rather than allocating maxEnd up front, so a header
	// that claims more data than the file has costs only what the file holds.
	data, err := io.ReadAll(io.LimitReader(r, int64(maxEnd)+1))
	if err != nil {
		return nil, fmt.Errorf("load edge topology: read data: %w", err)
	}
	if uint64(len(data)) < maxEnd {
		return nil, fmt.Errorf("load edge topology: data is %d bytes, the header describes %d", len(data), maxEnd)
	}
	if uint64(len(data)) > maxEnd {
		return nil, errors.New("load edge topology: file has trailing data")
	}
	tensors, err := loadSafeTensorData(entries, data)
	if err != nil {
		return nil, fmt.Errorf("load edge topology: decode SafeTensors: %w", err)
	}

	nodes := tensors["nodes"].int64Data[0]
	if nodes <= 0 || nodes > math.MaxInt32 {
		return nil, fmt.Errorf("load edge topology: node count %d is outside 1..%d", nodes, math.MaxInt32)
	}
	stored := [2][]int64{tensors["sources"].int64Data, tensors["targets"].int64Data}
	converted := [2][]int{make([]int, len(stored[0])), make([]int, len(stored[1]))}
	for k, name := range []string{"sources", "targets"} {
		for e, v := range stored[k] {
			if v < 0 || v >= nodes {
				return nil, fmt.Errorf("load edge topology: %s[%d] = %d is outside 0..%d", name, e, v, nodes-1)
			}
			converted[k][e] = int(v)
		}
	}
	g, err := NewEdgeTopology(int(nodes), converted[0], converted[1])
	if err != nil {
		return nil, fmt.Errorf("load edge topology: %w", err)
	}
	return g, nil
}

// Equal reports whether g and other have the same node count and the same
// edges in the same order, so per-edge weights saved against one apply to the
// other. Where each copy lives is not compared. Two nil topologies are equal;
// a nil one equals no other.
func (g *EdgeTopology) Equal(other *EdgeTopology) bool {
	if g == nil || other == nil {
		return g == other
	}
	if g.nodes != other.nodes || len(g.sources) != len(other.sources) {
		return false
	}
	for e := range g.sources {
		if g.sources[e] != other.sources[e] || g.targets[e] != other.targets[e] {
			return false
		}
	}
	return true
}
