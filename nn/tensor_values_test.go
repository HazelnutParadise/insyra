package nn

import "testing"

// float32Values reads a float32 tensor for a test and fails the test when the
// tensor is nil or of another dtype.
func float32Values(tb testing.TB, tensor *Tensor) []float32 {
	tb.Helper()
	values, err := tensor.Float32Data()
	if err != nil {
		tb.Fatalf("Float32Data: %v", err)
	}
	return values
}
