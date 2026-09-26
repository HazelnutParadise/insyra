package wgpu

// ExactSumParityRowsForTest exposes the parity row set to the external test that compares the device with nn.EdgeSum.
var ExactSumParityRowsForTest = exactSumParityRows

// RunExactSumHarnessForTest exposes the exact-sum harness to the external test that compares the device with nn.EdgeSum.
var RunExactSumHarnessForTest = runExactSumHarness

// RunExactSumFlatForTest exposes the flattened exact-sum harness to the external throughput measurement.
var RunExactSumFlatForTest = runExactSumFlat
