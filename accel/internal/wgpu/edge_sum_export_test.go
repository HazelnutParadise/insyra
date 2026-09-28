package wgpu

import "context"

// EdgeSumCSRForTest mirrors edgeSumCSR for the external measurement tests.
type EdgeSumCSRForTest = edgeSumCSR

// NewEdgeSumCSRForTest exposes the edge-sum CSR builder to measurement tests.
var NewEdgeSumCSRForTest = newEdgeSumCSR

// EdgeSumSeparatedWGSLForTest exposes the separated edge-sum shader to measurement tests.
var EdgeSumSeparatedWGSLForTest = edgeSumSeparatedWGSL

// EdgeSumPlainWGSLForTest exposes the plain edge-sum shader to measurement tests.
var EdgeSumPlainWGSLForTest = edgeSumPlainWGSL

// RunEdgeSumFullUploadForTest runs edge sum with every input uploaded on each call.
func RunEdgeSumFullUploadForTest(ctx context.Context, wgsl string, csr edgeSumCSR, weights, values []float32, batch int) ([]float32, error) {
	return runEdgeSumPrototype(ctx, wgsl, csr, weights, values, batch)
}

// EdgeSumResidentForTest exposes resident edge-sum state to measurement tests.
type EdgeSumResidentForTest struct{ r *edgeSumResident }

// NewEdgeSumResidentForTest creates resident edge-sum state for measurement tests.
func NewEdgeSumResidentForTest(csr edgeSumCSR) (*EdgeSumResidentForTest, error) {
	resident, err := newEdgeSumResident(csr)
	if err != nil {
		return nil, err
	}
	return &EdgeSumResidentForTest{r: resident}, nil
}

// Run executes edge sum with resident topology and per-call weights and values.
func (r *EdgeSumResidentForTest) Run(ctx context.Context, wgsl string, weights, values []float32, batch int) ([]float32, error) {
	return r.r.run(ctx, wgsl, weights, values, batch)
}

// Release frees the resident topology buffers.
func (r *EdgeSumResidentForTest) Release() {
	r.r.Release()
}
