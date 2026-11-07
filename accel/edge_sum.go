package accel

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
	"github.com/HazelnutParadise/insyra/internal/edgecsr"
)

// DeviceEdgeTopology is an immutable directed edge list held on one device,
// grouped both by target and by source, for RoundedEdgeSum and its gradients.
// Edge e runs from sources[e] to targets[e]. It owns device memory until
// Release, which is idempotent.
type DeviceEdgeTopology struct {
	device                     Device
	nodes, edges               int
	targetOffsets, targetEdges *DeviceBuffer
	sourceOffsets, sourceEdges *DeviceBuffer
	sources, targets           *DeviceBuffer
}

// NewDeviceEdgeTopology validates the edge list and uploads it to device once.
// nodes must be at least 1, and every source and target must lie in
// [0, nodes).
func NewDeviceEdgeTopology(device Device, nodes int, sources, targets []int32) (*DeviceEdgeTopology, error) {
	if device.ID == "" {
		return nil, errors.New("accel: NewDeviceEdgeTopology needs a device; the host keeps nn.EdgeTopology itself")
	}
	if nodes < 1 || nodes > math.MaxInt32-1 {
		return nil, fmt.Errorf("accel: edge topology needs between 1 and %d nodes, got %d", math.MaxInt32-1, nodes)
	}
	if len(sources) != len(targets) {
		return nil, fmt.Errorf("accel: edge topology has %d sources and %d targets", len(sources), len(targets))
	}
	for e := range sources {
		if sources[e] < 0 || int(sources[e]) >= nodes {
			return nil, fmt.Errorf("accel: edge %d source %d is outside [0, %d)", e, sources[e], nodes)
		}
		if targets[e] < 0 || int(targets[e]) >= nodes {
			return nil, fmt.Errorf("accel: edge %d target %d is outside [0, %d)", e, targets[e], nodes)
		}
	}
	grouped := edgecsr.Group(nodes, sources, targets)
	t := &DeviceEdgeTopology{device: device, nodes: nodes, edges: len(sources)}
	parts := []struct {
		dst  **DeviceBuffer
		bits []int32
	}{
		{&t.targetOffsets, grouped.TargetOffsets}, {&t.targetEdges, grouped.TargetEdges},
		{&t.sourceOffsets, grouped.SourceOffsets}, {&t.sourceEdges, grouped.SourceEdges},
		{&t.sources, sources}, {&t.targets, targets},
	}
	for _, part := range parts {
		bits := make([]uint32, len(part.bits))
		for i, v := range part.bits {
			bits[i] = uint32(v)
		}
		buffer, err := NewBitsBuffer(device, bits)
		if err != nil {
			t.Release()
			return nil, err
		}
		*part.dst = buffer
	}
	return t, nil
}

// Device reports the device the topology lives on.
func (t *DeviceEdgeTopology) Device() Device { return t.device }

// Nodes returns the number of nodes.
func (t *DeviceEdgeTopology) Nodes() int { return t.nodes }

// Edges returns the number of edges.
func (t *DeviceEdgeTopology) Edges() int { return t.edges }

// Released reports whether Release has been called.
func (t *DeviceEdgeTopology) Released() bool {
	return t == nil || t.targetOffsets == nil || t.targetOffsets.Released()
}

// Release frees the topology's device memory.
func (t *DeviceEdgeTopology) Release() {
	if t == nil {
		return
	}
	for _, b := range []*DeviceBuffer{t.targetOffsets, t.targetEdges, t.sourceOffsets, t.sourceEdges, t.sources, t.targets} {
		b.Release()
	}
}

// edgeOperands checks the topology and the buffers an edge-sum operation reads:
// a live topology, one device for all, weights with one value per edge when
// given, and node buffers shaped [batch,nodes].
func (t *DeviceEdgeTopology) edgeOperands(op string, batch int, weights *DeviceBuffer, nodeBuffers ...*DeviceBuffer) error {
	if t.Released() {
		return fmt.Errorf("accel: %s: the edge topology was released", op)
	}
	if batch < 0 || batch > math.MaxUint32/t.nodes {
		return fmt.Errorf("accel: %s: batch %d is outside [0, %d]", op, batch, math.MaxUint32/t.nodes)
	}
	buffers := nodeBuffers
	if weights != nil {
		buffers = append([]*DeviceBuffer{weights}, nodeBuffers...)
	}
	for _, b := range buffers {
		if b == nil || b.Released() {
			return fmt.Errorf("accel: %s: an operand is nil or released", op)
		}
		if b.device.ID != t.device.ID {
			return fmt.Errorf("accel: %s: operand on %s and topology on %s", op, b.device.ID, t.device.ID)
		}
	}
	if weights != nil && weights.Len() != t.edges {
		return fmt.Errorf("accel: %s: %d weights for %d edges", op, weights.Len(), t.edges)
	}
	for _, b := range nodeBuffers {
		if b.Len() != batch*t.nodes {
			return fmt.Errorf("accel: %s: a node operand holds %d values, want %d by %d", op, b.Len(), batch, t.nodes)
		}
	}
	return nil
}

// RoundedEdgeSum returns out[b,t], the sum of weights[e]*values[b,sources[e]]
// over every edge e into t, for values shaped [batch,nodes]. Each output is the
// exact sum of its products rounded once to nearest-even float32, the bits
// nn.EdgeSum gives on the host in every precision; a node with no incoming
// edge is +0.
func RoundedEdgeSum(t *DeviceEdgeTopology, weights, values *DeviceBuffer, batch int) (*DeviceBuffer, error) {
	if err := t.edgeOperands("RoundedEdgeSum", batch, weights, values); err != nil {
		return nil, err
	}
	out, err := wgpu.EdgeSumRounded(t.targetOffsets.inner, t.targetEdges.inner, t.sources.inner, weights.inner, values.inner, t.nodes, batch)
	if err != nil {
		return nil, residentFailure(err)
	}
	return values.wrap(out)
}

// RoundedEdgeSumGradValues returns the gradient of RoundedEdgeSum with respect
// to values: for each b and node s, the sum of weights[e]*upstream[b,targets[e]]
// over every edge e leaving s, rounded once.
func RoundedEdgeSumGradValues(t *DeviceEdgeTopology, weights, upstream *DeviceBuffer, batch int) (*DeviceBuffer, error) {
	if err := t.edgeOperands("RoundedEdgeSumGradValues", batch, weights, upstream); err != nil {
		return nil, err
	}
	out, err := wgpu.EdgeSumRounded(t.sourceOffsets.inner, t.sourceEdges.inner, t.targets.inner, weights.inner, upstream.inner, t.nodes, batch)
	if err != nil {
		return nil, residentFailure(err)
	}
	return upstream.wrap(out)
}

// RoundedEdgeSumGradWeights returns the gradient of RoundedEdgeSum with respect
// to the weights: for each edge e, the sum over the batch of
// upstream[b,targets[e]]*values[b,sources[e]], rounded once.
func RoundedEdgeSumGradWeights(t *DeviceEdgeTopology, values, upstream *DeviceBuffer, batch int) (*DeviceBuffer, error) {
	if err := t.edgeOperands("RoundedEdgeSumGradWeights", batch, nil, values, upstream); err != nil {
		return nil, err
	}
	out, err := wgpu.EdgeSumWeightsRounded(t.sources.inner, t.targets.inner, values.inner, upstream.inner, t.nodes, batch)
	if err != nil {
		return nil, residentFailure(err)
	}
	return values.wrap(out)
}
