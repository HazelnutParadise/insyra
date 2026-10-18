package nn

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

// dotProductMatMul is the per-element reference: each output is one running
// sum that starts at zero and adds its products in inner-index order.
func dotProductMatMul(a []float32, b []float32, rows, inner, columns int) []float32 {
	out := make([]float32, rows*columns)
	for row := 0; row < rows; row++ {
		for column := 0; column < columns; column++ {
			var sum float32
			for index := 0; index < inner; index++ {
				sum += a[row*inner+index] * b[index*columns+column]
			}
			out[row*columns+column] = sum
		}
	}
	return out
}

func randomMatMulData(r *rand.Rand, size int, special bool) []float32 {
	data := make([]float32, size)
	specials := []float32{0, float32(math.Copysign(0, -1)), float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.NaN()), math.SmallestNonzeroFloat32, math.MaxFloat32}
	for index := range data {
		if special && r.IntN(8) == 0 {
			data[index] = specials[r.IntN(len(specials))]
			continue
		}
		data[index] = float32(r.NormFloat64())
	}
	return data
}

// TestMatMul2DKeepsTheDotProductOrder checks every bit of the 2-D kernel,
// serial and parallel, against the per-element reference, including empty
// inner dimensions and inputs holding signed zeros, infinities, NaN and
// subnormal products. A NaN output only has to be NaN: Go does not specify
// which NaN an addition of two NaNs returns, and the compiler may order the
// operands of one addition either way.
func TestMatMul2DKeepsTheDotProductOrder(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	shapes := [][3]int{{1, 1, 1}, {1, 784, 28}, {2, 24, 11}, {3, 0, 4}, {5, 7, 1}, {64, 64, 64}, {129, 33, 17}}
	for _, special := range []bool{false, true} {
		for _, shape := range shapes {
			rows, inner, columns := shape[0], shape[1], shape[2]
			left := randomMatMulData(r, rows*inner, special)
			right := randomMatMulData(r, inner*columns, special)
			want := dotProductMatMul(left, right, rows, inner, columns)
			a := mustTestTensor(t, []int{rows, inner}, left)
			b := mustTestTensor(t, []int{inner, columns}, right)
			for _, workers := range []int{1, 4} {
				t.Run(fmt.Sprintf("%dx%dx%d/special=%v/workers=%d", rows, inner, columns, special, workers), func(t *testing.T) {
					got, err := matMul2DWithWorkers(a, b, workers)
					if err != nil {
						t.Fatal(err)
					}
					for index, value := range want {
						if math.IsNaN(float64(value)) && math.IsNaN(float64(got.data[index])) {
							continue
						}
						if math.Float32bits(got.data[index]) != math.Float32bits(value) {
							t.Fatalf("output %d = %v (%#08x), want %v (%#08x)", index, got.data[index], math.Float32bits(got.data[index]), value, math.Float32bits(value))
						}
					}
				})
			}
		}
	}
}

func BenchmarkMatMul2D(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 2))
	for _, shape := range [][3]int{{1, 784, 28}, {2, 24, 11}, {64, 784, 28}, {256, 256, 256}} {
		rows, inner, columns := shape[0], shape[1], shape[2]
		left, err := NewTensor([]int{rows, inner}, randomMatMulData(r, rows*inner, false))
		if err != nil {
			b.Fatal(err)
		}
		right, err := NewTensor([]int{inner, columns}, randomMatMulData(r, inner*columns, false))
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("%dx%dx%d", rows, inner, columns), func(b *testing.B) {
			for b.Loop() {
				if _, err := matMul2DWithWorkers(left, right, 0); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(rows*inner*columns), "ns/MAC")
		})
	}
}
