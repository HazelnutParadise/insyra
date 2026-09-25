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
	dValues, err := newZeroFloat32Tensor(values.shape)
	if err != nil {
		return nil, err
	}
	for b := 0; b < batch; b++ {
		for s := 0; s < n; s++ {
			acc := float32(0)
			for i := int(topology.sourceOffsets[s]); i < int(topology.sourceOffsets[s+1]); i++ {
				e := int(topology.sourceEdges[i])
				acc += float32(weights.data[e] * upstream.data[b*n+int(topology.targets[e])]) // explicit conversion: no fused multiply-add
			}
			dValues.data[b*n+s] = acc
		}
	}
	dWeights, err := newZeroFloat32Tensor([]int{topology.Edges()})
	if err != nil {
		return nil, err
	}
	for e := 0; e < topology.Edges(); e++ {
		acc := float32(0)
		for b := 0; b < batch; b++ {
			acc += float32(upstream.data[b*n+int(topology.targets[e])] * values.data[b*n+int(topology.sources[e])]) // explicit conversion: no fused multiply-add
		}
		dWeights.data[e] = acc
	}
	return []*Tensor{dWeights, dValues}, nil
}
