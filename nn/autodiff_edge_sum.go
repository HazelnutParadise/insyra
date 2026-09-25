package nn

import "fmt"

// EdgeSum runs EdgeSum and records its reverse rule. The gradient of
// values[..., s] sums weights[e]*upstream[..., targets[e]] over the edges
// leaving s; the gradient of weights[e] sums
// upstream[b, targets[e]]*values[b, sources[e]] over the batch. The output and
// every gradient are the exact sums of their products rounded once to the
// nearest float32, with ties to even, so results do not depend on edge order, batch
// order, or worker count. An exact zero is +0. A NaN operand, zero times
// infinity, or infinite products of both signs produce NaN; otherwise an
// infinite product determines the result.
func (t *Tape) EdgeSum(topology *EdgeTopology, weights, values *Tensor) (*Tensor, error) {
	output, err := EdgeSum(topology, weights, values)
	if err != nil {
		return nil, err
	}
	t.record("EdgeSum", []*Tensor{weights, values}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return edgeSumVJP(topology, weights, values, upstream)
	})
	return output, nil
}

func edgeSumVJP(topology *EdgeTopology, weights, values, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "edge sum upstream"); err != nil {
		return nil, err
	}
	if !sameShape(upstream.shape, values.shape) {
		return nil, fmt.Errorf("edge sum upstream shape %v does not match output shape %v", upstream.shape, values.shape)
	}
	n := topology.Nodes()
	batch := 1
	if len(values.shape) == 2 {
		batch = values.shape[0]
	}
	return edgeSumVJPWith(topology, weights, values, upstream, batch, edgeSumWorkers(batch, n, topology.Edges()))
}

func edgeSumVJPWith(topology *EdgeTopology, weights, values, upstream *Tensor, batch, workers int) ([]*Tensor, error) {
	n := topology.Nodes()
	dValues, err := newZeroFloat32Tensor(values.shape)
	if err != nil {
		return nil, err
	}
	parallelFor(batch*n, workers, func(start, end int) {
		var acc exactAccumulator
		for i := start; i < end; i++ {
			b, s := i/n, i%n
			acc.reset()
			for edgeIndex := int(topology.sourceOffsets[s]); edgeIndex < int(topology.sourceOffsets[s+1]); edgeIndex++ {
				e := int(topology.sourceEdges[edgeIndex])
				acc.addProduct(weights.data[e], upstream.data[b*n+int(topology.targets[e])])
			}
			dValues.data[i] = acc.float32()
		}
	})
	dWeights, err := newZeroFloat32Tensor([]int{topology.Edges()})
	if err != nil {
		return nil, err
	}
	parallelFor(topology.Edges(), workers, func(start, end int) {
		var acc exactAccumulator
		for e := start; e < end; e++ {
			acc.reset()
			for b := 0; b < batch; b++ {
				acc.addProduct(upstream.data[b*n+int(topology.targets[e])], values.data[b*n+int(topology.sources[e])])
			}
			dWeights.data[e] = acc.float32()
		}
	})
	return []*Tensor{dWeights, dValues}, nil
}
