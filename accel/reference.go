package accel

// CPU reference implementations. Every device operation ships one, because
// default-on execution is gated on the device result being bit-identical to it
// on the running platform.

// squaredDistancesReference computes the squared Euclidean distance from every
// row to every query point, query-major.
//
// The accumulation is written `acc + diff*diff` on purpose, and must not be
// "corrected" to `acc + float32(diff*diff)`. Measured on Apple M3: Metal
// contracts multiply-add into a fused instruction, Go does the same on arm64,
// and the two contract identically — so the fused form is bit-identical to the
// device across 4096 outputs while the explicitly-rounded form differs in 1137
// of them by up to 2 ulp. The precaution that looks missing here is what would
// break parity.
//
// columns is column-major: columns[c][r] is row r of column c.
func squaredDistancesReference(columns [][]float32, queries [][]float32, rows int) []float32 {
	out := make([]float32, len(queries)*rows)
	for q, query := range queries {
		base := q * rows
		for r := 0; r < rows; r++ {
			var acc float32
			for c := range columns {
				diff := columns[c][r] - query[c]
				acc = acc + diff*diff
			}
			out[base+r] = acc
		}
	}
	return out
}
