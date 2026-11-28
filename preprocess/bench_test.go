package preprocess

import (
	"fmt"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// benchTable is the 100,000-row table the preprocess benchmarks fit and
// transform: two numeric columns with a few missing cells and one column of
// ten categories.
func benchTable(n int) *insyra.DataTable {
	x := make([]any, n)
	y := make([]any, n)
	c := make([]any, n)
	for i := 0; i < n; i++ {
		x[i] = float64(i%997) * 1.5
		y[i] = float64((i*7)%613) - 300
		if i%11 == 0 {
			y[i] = nil
		}
		c[i] = fmt.Sprintf("cat%d", i%10)
	}
	return insyra.NewDataTable(
		insyra.NewDataList(x...).SetName("x"),
		insyra.NewDataList(y...).SetName("y"),
		insyra.NewDataList(c...).SetName("c"),
	)
}

func BenchmarkStandardScale100k(b *testing.B) {
	dt := benchTable(100000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := StandardScale(dt, insyra.Name("x"), insyra.Name("y")); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMinMaxScale100k(b *testing.B) {
	dt := benchTable(100000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := MinMaxScale(dt, 0, 1, insyra.Name("x"), insyra.Name("y")); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOneHotEncode100k(b *testing.B) {
	dt := benchTable(100000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := OneHotEncode(dt, OneHotOptions{Columns: []any{insyra.Name("c")}}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLabelEncode100k(b *testing.B) {
	dt := benchTable(100000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := LabelEncode(dt, LabelOptions{Column: insyra.Name("c")}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSimpleImputer100k(b *testing.B) {
	dt := benchTable(100000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeMean}).FitTransform(dt, insyra.Name("y")); err != nil {
			b.Fatal(err)
		}
	}
}
