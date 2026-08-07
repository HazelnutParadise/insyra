package insyra

import "testing"

const issue191Expression = "SQRT(A*A + B*B) * 2 + A / B - 1"

func benchmarkIssue191(b *testing.B, rows int) {
	b.Helper()
	a := make([]any, rows)
	bCol := make([]any, rows)
	for i := 0; i < rows; i++ {
		a[i] = float64(i + 1)
		bCol[i] = float64(i + 2)
	}
	b.ResetTimer()
	for b.Loop() {
		dt := NewDataTable(NewDataList(a...), NewDataList(bCol...))
		dt.AddColUsingCCL("C", issue191Expression)
		if err := dt.Err(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIssue191_AddColUsingCCL_10k(b *testing.B) {
	benchmarkIssue191(b, 10_000)
}

func BenchmarkIssue191_AddColUsingCCL_100k(b *testing.B) {
	benchmarkIssue191(b, 100_000)
}
