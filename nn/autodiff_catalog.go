package nn

import "fmt"

// Embedding looks up rows from a [vocab, dim] table for int64 indices with
// shape [N] or [N,S]. Its VJP scatter-adds repeated index contributions.
func (t *Tape) Embedding(table, indices *Tensor) (*Tensor, error) {
	if err := requireFloat32(table, "embedding table"); err != nil {
		return nil, err
	}
	if indices == nil || indices.dtype != DTypeInt64 {
		return nil, fmt.Errorf("embedding indices must be int64")
	}
	if len(table.shape) != 2 {
		return nil, fmt.Errorf("embedding table must be 2-D, got shape %v", table.shape)
	}
	if len(indices.shape) != 1 && len(indices.shape) != 2 {
		return nil, fmt.Errorf("embedding indices must be 1-D or 2-D, got shape %v", indices.shape)
	}
	for position, index := range indices.int64Data {
		if index < 0 || index >= int64(table.shape[0]) {
			return nil, fmt.Errorf("embedding index %d at position %d is out of range [0, %d)", index, position, table.shape[0])
		}
	}
	outputShape := append(append([]int(nil), indices.shape...), table.shape[1])
	output, err := newZeroFloat32Tensor(outputShape)
	if err != nil {
		return nil, err
	}
	for position, index := range indices.int64Data {
		start := int(index) * table.shape[1]
		copy(output.data[position*table.shape[1]:(position+1)*table.shape[1]], table.data[start:start+table.shape[1]])
	}
	t.record("Embedding", []*Tensor{table}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{embeddingVJP(table, indices, upstream)}, nil
	})
	return output, nil
}

// EmbeddingLookup is an argument-order alias for callers that prefer indices
// first, matching the layer's Forward input order.
func (t *Tape) EmbeddingLookup(indices, table *Tensor) (*Tensor, error) {
	return t.Embedding(table, indices)
}

func embeddingVJP(table, indices, upstream *Tensor) *Tensor {
	gradient, _ := newZeroFloat32Tensor(table.shape)
	dim := table.shape[1]
	for position, index := range indices.int64Data {
		for feature := 0; feature < dim; feature++ {
			gradient.data[int(index)*dim+feature] += upstream.data[position*dim+feature]
		}
	}
	return gradient
}
