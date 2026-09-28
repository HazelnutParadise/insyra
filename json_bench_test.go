// BenchmarkJSONPaths is the benchmark AGENTS.md's "One JSON library" rule uses
// to compare a candidate JSON library against the one insyra reads and writes
// with. It measures insyra's own two paths — ToJSON_Bytes and ReadJSON — over a
// fixed 100,000 x 5 table built from a fixed seed, so two runs of two libraries
// are comparable.
//
// To compare two libraries, run
//
//	go test -run '^$' -bench BenchmarkJSONPaths -count 5 .
//
// on each and take each sub-benchmark's fastest of the five: the seeds fix the
// data, so the numbers differ only by the library and by machine noise.

package insyra

import (
	"fmt"
	"math/rand"
	"testing"
)

const jsonBenchRows = 100_000

// newJSONBenchTable builds the fixed table every sub-benchmark reads and writes.
// The seed is fixed so the data is identical across runs and across libraries.
func newJSONBenchTable() *DataTable {
	rng := rand.New(rand.NewSource(1))
	cities := []string{"台北", "Tokyo", "Paris"}

	ids := make([]any, jsonBenchRows)
	values := make([]any, jsonBenchRows)
	labels := make([]any, jsonBenchRows)
	flags := make([]any, jsonBenchRows)
	scores := make([]any, jsonBenchRows)

	for i := 0; i < jsonBenchRows; i++ {
		ids[i] = int64(rng.Intn(1_000_000))
		values[i] = rng.NormFloat64() * 1000
		labels[i] = fmt.Sprintf("name-%d-%s", i, cities[i%3])
		flags[i] = i%2 == 0
		// A nil every tenth row, so the read path carries a JSON null.
		if i%10 == 0 {
			scores[i] = nil
		} else {
			scores[i] = rng.Float64()
		}
	}

	return NewDataTable(
		NewDataList(ids...).SetName("id"),
		NewDataList(values...).SetName("value"),
		NewDataList(labels...).SetName("label"),
		NewDataList(flags...).SetName("flag"),
		NewDataList(scores...).SetName("score"),
	)
}

func BenchmarkJSONPaths(b *testing.B) {
	// Built once, outside every sub-benchmark, so no timing covers construction.
	dt := newJSONBenchTable()
	encoded := dt.ToJSON_Bytes(true)

	b.Run("ToJSON_Bytes", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(encoded)))
		for b.Loop() {
			_ = dt.ToJSON_Bytes(true)
		}
	})

	b.Run("ReadJSON", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(encoded)))
		for b.Loop() {
			if _, err := ReadJSON(encoded); err != nil {
				b.Fatal(err)
			}
		}
	})
}
