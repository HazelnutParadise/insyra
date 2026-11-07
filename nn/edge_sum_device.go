package nn

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/accel"
)

// To returns the topology with a copy on device, for Tape.EdgeSum on tensors
// there. The zero Device stands for the host and returns the topology itself.
// The copy owns device memory until Release; moving a topology already on a
// device makes a new copy and leaves the old one alone.
func (g *EdgeTopology) To(device accel.Device) (*EdgeTopology, error) {
	if g == nil {
		return nil, fmt.Errorf("edge topology is nil")
	}
	if device.ID == "" {
		return g, nil
	}
	resident, err := accel.NewDeviceEdgeTopology(device, g.nodes, g.sources, g.targets)
	if err != nil {
		return nil, err
	}
	moved := *g
	moved.device = resident
	return &moved, nil
}

// Device reports where the topology's copy lives. The zero Device is the host.
func (g *EdgeTopology) Device() accel.Device {
	if g == nil || g.device == nil {
		return accel.Device{}
	}
	return g.device.Device()
}

// Release frees the device copy To made. It is idempotent, and does nothing
// for a topology on the host.
func (g *EdgeTopology) Release() {
	if g != nil {
		g.device.Release()
	}
}

// edgeSumResident runs EdgeSum on weights and values on a device and records a
// reverse rule whose two gradients stay there.
func (t *Tape) edgeSumResident(topology *EdgeTopology, weights, values *Tensor) (*Tensor, error) {
	if topology == nil {
		return nil, fmt.Errorf("edge sum topology is nil")
	}
	device := values.Device().ID
	if topology.device == nil {
		return nil, fmt.Errorf("edge sum operands live on the device %s and the topology on the host; move the topology with To", device)
	}
	if topology.Device().ID != device {
		return nil, fmt.Errorf("edge sum operands live on the device %s and the topology on %s", device, topology.Device().ID)
	}
	if weights.dtype != DTypeFloat32 || values.dtype != DTypeFloat32 {
		return nil, fmt.Errorf("edge sum on a device needs float32 weights and values, got %s and %s", dtypeName(weights.dtype), dtypeName(values.dtype))
	}
	edges, n := topology.Edges(), topology.Nodes()
	if len(weights.shape) != 1 || weights.shape[0] != edges {
		return nil, fmt.Errorf("edge sum weights must have shape [%d], got %v", edges, weights.shape)
	}
	batch := 1
	switch {
	case len(values.shape) == 1 && values.shape[0] == n:
	case len(values.shape) == 2 && values.shape[1] == n:
		batch = values.shape[0]
	default:
		return nil, fmt.Errorf("edge sum values must have shape [%d] or [B, %d], got %v", n, n, values.shape)
	}
	buffer, err := accel.RoundedEdgeSum(topology.device, weights.device, values.device, batch)
	if err != nil {
		return nil, err
	}
	output, err := t.ownResident(values.shape, buffer)
	if err != nil {
		return nil, err
	}
	t.record("EdgeSum", []*Tensor{weights, values}, output, func(upstream *Tensor) ([]*Tensor, error) {
		if !upstream.onDevice() || upstream.Device().ID != device {
			return nil, fmt.Errorf("edge sum upstream must be on the device %s", device)
		}
		if !sameShape(upstream.shape, values.shape) {
			return nil, fmt.Errorf("edge sum upstream shape %v does not match output shape %v", upstream.shape, values.shape)
		}
		dWeightsBuffer, err := accel.RoundedEdgeSumGradWeights(topology.device, values.device, upstream.device, batch)
		if err != nil {
			return nil, err
		}
		dWeights, err := t.ownResident([]int{edges}, dWeightsBuffer)
		if err != nil {
			return nil, err
		}
		dValuesBuffer, err := accel.RoundedEdgeSumGradValues(topology.device, weights.device, upstream.device, batch)
		if err != nil {
			return nil, err
		}
		dValues, err := t.ownResident(values.shape, dValuesBuffer)
		if err != nil {
			return nil, err
		}
		return []*Tensor{dWeights, dValues}, nil
	})
	return output, nil
}

// ownResident wraps a buffer an operation returned as a tensor the tape owns.
// newResidentTensor releases the buffer if the wrapping fails.
func (t *Tape) ownResident(shape []int, buffer *accel.DeviceBuffer) (*Tensor, error) {
	tensor, err := newResidentTensor(shape, buffer)
	if err != nil {
		return nil, err
	}
	return t.own(tensor), nil
}
