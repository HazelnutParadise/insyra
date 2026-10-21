package nn

import "fmt"

// matMulRounded multiplies two 2-D float32 tensors, reading a as its
// transpose when transA is true and b as its transpose when transB is true,
// without copying either. Every output element is the exact sum of its
// products rounded once to nearest-even float32, so the result does not
// depend on the order the products are added in, on the number of workers, or
// on the platform. This is the CPU reference for PrecisionFloat32Rounded.
//
// Expected results (the tests pin each one):
//   - the output is shaped [rows(a'), cols(b')] where a' and b' are the operands
//     after the optional transpose; incompatible inner dimensions, a non-2-D or
//     non-float32 operand, or a nil operand is an error naming the shapes;
//   - an element equals the math/big exact dot product rounded once to float32
//     on random data, on cancelling data, on subnormals, at the overflow boundary,
//     with NaN and infinities (any NaN input gives a NaN, inf-inf gives NaN),
//     and for signed zeros (an exactly-zero sum is +0, as an accumulation into a
//     zeroed output);
//   - an inner dimension of zero gives an all-zero output of the right shape;
//   - transA/transB give bit-identical results to multiplying explicitly
//     transposed copies;
//   - one worker and every worker give identical bits;
//   - the operands are never modified.
func matMulRounded(a, b *Tensor, transA, transB bool) (*Tensor, error) {
	return matMulRoundedWithWorkers(a, b, transA, transB, 0)
}

// matMulRoundedWithWorkers is matMulRounded with the number of goroutines
// that share the output rows chosen by the caller; zero or less picks the
// same count the default MatMul would. The result must not depend on it.
func matMulRoundedWithWorkers(a, b *Tensor, transA, transB bool, workers int) (*Tensor, error) {
	shapeA, shapeB := "<nil>", "<nil>"
	if a != nil {
		shapeA = fmt.Sprint(a.shape)
	}
	if b != nil {
		shapeB = fmt.Sprint(b.shape)
	}
	shapeError := func(reason string) error {
		return fmt.Errorf("matmul rounded shapes %s and %s: %s", shapeA, shapeB, reason)
	}
	if a == nil || b == nil {
		return nil, shapeError("operands must not be nil")
	}
	if len(a.shape) != 2 || len(b.shape) != 2 {
		return nil, shapeError("operands must both be 2-D")
	}
	if a.dtype != DTypeFloat32 || b.dtype != DTypeFloat32 {
		return nil, shapeError("operands must both have float32 dtype")
	}

	rows, innerA := a.shape[0], a.shape[1]
	if transA {
		rows, innerA = a.shape[1], a.shape[0]
	}
	innerB, columns := b.shape[0], b.shape[1]
	if transB {
		innerB, columns = b.shape[1], b.shape[0]
	}
	if innerA != innerB {
		return nil, shapeError(fmt.Sprintf("inner dimensions %d and %d are incompatible", innerA, innerB))
	}

	result, err := newZeroFloat32Tensor([]int{rows, columns})
	if err != nil {
		return nil, err
	}
	if workers <= 0 {
		workers = parallelWorkerCountForMACs(rows, columns, innerA)
	}
	left, right, output := a.data, b.data, result.data
	aColumns, bColumns := a.shape[1], b.shape[1]
	parallelFor(rows, workers, func(start, end int) {
		var fast float64ProductSum
		var exact exactAccumulator
		for row := start; row < end; row++ {
			for column := 0; column < columns; column++ {
				fast.reset()
				for index := 0; index < innerA; index++ {
					leftIndex := row*innerA + index
					if transA {
						leftIndex = index*aColumns + row
					}
					rightIndex := index*columns + column
					if transB {
						rightIndex = column*bColumns + index
					}
					fast.add(left[leftIndex], right[rightIndex])
				}
				if value, ok := fast.result(); ok {
					output[row*columns+column] = value
					continue
				}
				exact.reset()
				for index := 0; index < innerA; index++ {
					leftIndex := row*innerA + index
					if transA {
						leftIndex = index*aColumns + row
					}
					rightIndex := index*columns + column
					if transB {
						rightIndex = column*bColumns + index
					}
					exact.addProduct(left[leftIndex], right[rightIndex])
				}
				output[row*columns+column] = exact.float32()
			}
		}
	})
	return result, nil
}
