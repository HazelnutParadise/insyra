package nn

import "fmt"

// EdgeSum runs EdgeSum and records its reverse rule. The gradient of
// values[..., s] sums weights[e]*upstream[..., targets[e]] over the edges
// leaving s in ascending edge index; the gradient of weights[e] sums
// upstream[b, targets[e]]*values[b, sources[e]] over the batch in ascending
// batch index. Every product is rounded to float32 before it is added.
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
		for i := start; i < end; i++ {
			b, s := i/n, i%n
			acc := float32(0)
			for edgeIndex := int(topology.sourceOffsets[s]); edgeIndex < int(topology.sourceOffsets[s+1]); edgeIndex++ {
				e := int(topology.sourceEdges[edgeIndex])
				acc += float32(weights.data[e] * upstream.data[b*n+int(topology.targets[e])]) // explicit conversion: no fused multiply-add
			}
			dValues.data[i] = acc
		}
	})
	dWeights, err := newZeroFloat32Tensor([]int{topology.Edges()})
	if err != nil {
		return nil, err
	}
	parallelFor(topology.Edges(), workers, func(start, end int) {
		for e := start; e < end; e++ {
			acc := float32(0)
			for b := 0; b < batch; b++ {
				acc += float32(upstream.data[b*n+int(topology.targets[e])] * values.data[b*n+int(topology.sources[e])]) // explicit conversion: no fused multiply-add
			}
			dWeights.data[e] = acc
		}
	})
	return []*Tensor{dWeights, dValues}, nil
}
