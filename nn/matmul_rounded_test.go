package nn

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// matMulRoundedOracle uses exact rationals for every finite product and sum,
// then asks big.Float to perform the single final nearest-even float32 round.
func matMulRoundedOracle(a, b *Tensor, transA, transB bool) []float32 {
	ar, ac := a.shape[0], a.shape[1]
	br, bc := b.shape[0], b.shape[1]
	if transA {
		ar, ac = ac, ar
	}
	if transB {
		br, bc = bc, br
	}
	out := make([]float32, ar*bc)
	for i := 0; i < ar; i++ {
		for j := 0; j < bc; j++ {
			sum := new(big.Rat)
			posInf, negInf, nan := false, false, false
			for k := 0; k < ac; k++ {
				x := matrixAt(a, i, k, transA)
				y := matrixAt(b, k, j, transB)
				if math.IsNaN(float64(x)) || math.IsNaN(float64(y)) {
					nan = true
					continue
				}
				if math.IsInf(float64(x), 0) || math.IsInf(float64(y), 0) {
					if x == 0 || y == 0 {
						nan = true
						continue
					}
					negative := math.Signbit(float64(x)) != math.Signbit(float64(y))
					if negative {
						negInf = true
					} else {
						posInf = true
					}
					continue
				}
				rx := new(big.Rat).SetFloat64(float64(x))
				ry := new(big.Rat).SetFloat64(float64(y))
				sum.Add(sum, new(big.Rat).Mul(rx, ry))
			}
			if nan || posInf && negInf {
				out[i*bc+j] = float32(math.NaN())
				continue
			}
			if posInf {
				out[i*bc+j] = float32(math.Inf(1))
				continue
			}
			if negInf {
				out[i*bc+j] = float32(math.Inf(-1))
				continue
			}
			if sum.Sign() == 0 {
				out[i*bc+j] = 0
				continue
			}
			f, _ := new(big.Float).SetPrec(1024).SetMode(big.ToNearestEven).SetRat(sum).Float32()
			out[i*bc+j] = f
		}
	}
	return out
}

func matrixAt(t *Tensor, row, col int, trans bool) float32 {
	if trans {
		row, col = col, row
	}
	return t.data[row*t.shape[1]+col]
}

func assertRoundedBits(t *testing.T, got, want []float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("output length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if math.IsNaN(float64(got[i])) && math.IsNaN(float64(want[i])) {
			continue
		}
		if gb, wb := math.Float32bits(got[i]), math.Float32bits(want[i]); gb != wb {
			t.Fatalf("output[%d] bits = %#08x, want %#08x (values %g, %g)", i, gb, wb, got[i], want[i])
		}
	}
}

func roundedTensor(t *testing.T, shape []int, data []float32) *Tensor {
	t.Helper()
	x, err := NewTensor(shape, data)
	if err != nil {
		t.Fatalf("NewTensor(%v): %v", shape, err)
	}
	return x
}

func TestMatMulRoundedExactOracle(t *testing.T) {
	nextSubnormal := math.Float32frombits(1)
	tests := []struct {
		name   string
		as, bs []int
		a, b   []float32
	}{
		{"random", []int{3, 4}, []int{4, 2}, []float32{.25, -3, 1.5, 8, -2, .125, 4, -.5, 7, 2, -1, 3}, []float32{1, 2, -3, .5, 4, -2, .25, 8}},
		{"cancellation", []int{1, 4}, []int{4, 1}, []float32{1e20, -1e20, 3, -3}, []float32{1, 1, 1, 1}},
		{"subnormal", []int{1, 2}, []int{2, 1}, []float32{nextSubnormal, nextSubnormal}, []float32{1, 1}},
		{"overflow boundary", []int{1, 2}, []int{2, 1}, []float32{math.MaxFloat32, float32(math.Ldexp(1, 102))}, []float32{1, 1}},
		{"NaN and infinities", []int{1, 4}, []int{4, 1}, []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 2}, []float32{1, 1, 1, 0}},
		{"overflow midpoint rounds to even, which is infinity", []int{1, 2}, []int{2, 1}, []float32{math.MaxFloat32, float32(math.Ldexp(1, 103))}, []float32{1, 1}},
		{"positive infinity alone", []int{1, 2}, []int{2, 1}, []float32{float32(math.Inf(1)), 2}, []float32{3, 5}},
		{"negative infinity alone", []int{1, 2}, []int{2, 1}, []float32{float32(math.Inf(-1)), 2}, []float32{3, 5}},
		{"infinity times zero", []int{1, 2}, []int{2, 1}, []float32{float32(math.Inf(1)), 2}, []float32{0, 5}},
		{"opposite infinities", []int{1, 2}, []int{2, 1}, []float32{float32(math.Inf(1)), float32(math.Inf(1))}, []float32{1, -1}},
		{"exactly zero from nonzero terms", []int{1, 2}, []int{2, 1}, []float32{1.5, -1.5}, []float32{2, 2}},
		{"signed zero", []int{1, 3}, []int{3, 1}, []float32{float32(math.Copysign(0, -1)), 0, -0.0}, []float32{1, -1, 3}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, b := roundedTensor(t, tc.as, tc.a), roundedTensor(t, tc.bs, tc.b)
			got, err := matMulRounded(a, b, false, false)
			if err != nil {
				t.Fatalf("matMulRounded: %v", err)
			}
			assertRoundedBits(t, got.data, matMulRoundedOracle(a, b, false, false))
		})
	}
}

func TestMatMulRoundedZeroInnerDimension(t *testing.T) {
	a, b := roundedTensor(t, []int{2, 0}, nil), roundedTensor(t, []int{0, 3}, nil)
	got, err := matMulRounded(a, b, false, false)
	if err != nil {
		t.Fatalf("matMulRounded: %v", err)
	}
	if !reflect.DeepEqual(got.shape, []int{2, 3}) {
		t.Fatalf("shape = %v, want [2 3]", got.shape)
	}
	assertRoundedBits(t, got.data, make([]float32, 6))
}

func TestMatMulRoundedTransposeFlags(t *testing.T) {
	for _, ta := range []bool{false, true} {
		for _, tb := range []bool{false, true} {
			t.Run(fmt.Sprintf("transA=%t_transB=%t", ta, tb), func(t *testing.T) {
				// The product is [2,3]x[3,4]; the stored shapes follow the flags.
				aShape, bShape := []int{2, 3}, []int{3, 4}
				if ta {
					aShape = []int{3, 2}
				}
				if tb {
					bShape = []int{4, 3}
				}
				aData, bData := make([]float32, 6), make([]float32, 12)
				for i := range aData {
					aData[i] = float32(i+1) / 3
				}
				for i := range bData {
					bData[i] = float32(12-i) / 7
				}
				a, b := roundedTensor(t, aShape, aData), roundedTensor(t, bShape, bData)
				got, err := matMulRounded(a, b, ta, tb)
				if err != nil {
					t.Fatalf("matMulRounded: %v", err)
				}
				assertRoundedBits(t, got.data, matMulRoundedOracle(a, b, ta, tb))
				aa, bb := a, b
				if ta {
					aa, err = Transpose(a)
					if err != nil {
						t.Fatal(err)
					}
				}
				if tb {
					bb, err = Transpose(b)
					if err != nil {
						t.Fatal(err)
					}
				}
				want, err := matMulRounded(aa, bb, false, false)
				if err != nil {
					t.Fatalf("explicit transpose multiply: %v", err)
				}
				assertRoundedBits(t, got.data, want.data)
			})
		}
	}
}

func TestMatMulRoundedWorkerSizeDeterminism(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for _, dims := range [][3]int{{2, 7, 3}, {64, 128, 48}} {
		m, k, n := dims[0], dims[1], dims[2]
		aData, bData := make([]float32, m*k), make([]float32, k*n)
		for i := range aData {
			aData[i] = float32(rng.Intn(2001)-1000) / 37
		}
		for i := range bData {
			bData[i] = float32(rng.Intn(2001)-1000) / 41
		}
		a, b := roundedTensor(t, []int{m, k}, aData), roundedTensor(t, []int{k, n}, bData)
		got, err := matMulRounded(a, b, false, false)
		if err != nil {
			t.Fatalf("matMulRounded %v: %v", dims, err)
		}
		assertRoundedBits(t, got.data, matMulRoundedOracle(a, b, false, false))
	}
}

func TestMatMulRoundedWorkerCountDoesNotChangeBits(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	m, k, n := 37, 211, 29
	aData, bData := make([]float32, m*k), make([]float32, k*n)
	for i := range aData {
		aData[i] = float32(rng.NormFloat64()) * float32(math.Ldexp(1, rng.Intn(40)-20))
	}
	for i := range bData {
		bData[i] = float32(rng.NormFloat64()) * float32(math.Ldexp(1, rng.Intn(40)-20))
	}
	a, b := roundedTensor(t, []int{m, k}, aData), roundedTensor(t, []int{k, n}, bData)
	one, err := matMulRoundedWithWorkers(a, b, false, false, 1)
	if err != nil {
		t.Fatalf("one worker: %v", err)
	}
	assertRoundedBits(t, one.data, matMulRoundedOracle(a, b, false, false))
	for _, workers := range []int{2, 3, 8, 64} {
		many, err := matMulRoundedWithWorkers(a, b, false, false, workers)
		if err != nil {
			t.Fatalf("%d workers: %v", workers, err)
		}
		assertRoundedBits(t, many.data, one.data)
	}
}

func TestMatMulRoundedDiffersFromTheFixedOrderSumWhereItShould(t *testing.T) {
	// 2^24 + 1 + 1 summed in order loses both ones (each add rounds back to
	// 2^24); the exact sum is 2^24 + 2, which is representable.
	a := roundedTensor(t, []int{1, 3}, []float32{16777216, 1, 1})
	b := roundedTensor(t, []int{3, 1}, []float32{1, 1, 1})
	got, err := matMulRounded(a, b, false, false)
	if err != nil {
		t.Fatalf("matMulRounded: %v", err)
	}
	if got.data[0] != 16777218 {
		t.Fatalf("result = %v, want 16777218", got.data[0])
	}
}

func TestMatMulRoundedSeesBelowAFloat64Midpoint(t *testing.T) {
	// 1 + 2^-24 is exactly halfway between two float32 values, and the 2^-80
	// term decides which way it rounds. A float64 total drops the 2^-80 term
	// and ties to even, which is wrong; the exact sum rounds up.
	for _, order := range [][]float32{
		{1, float32(math.Ldexp(1, -24)), float32(math.Ldexp(1, -80))},
		{float32(math.Ldexp(1, -80)), float32(math.Ldexp(1, -24)), 1},
	} {
		a := roundedTensor(t, []int{1, 3}, order)
		b := roundedTensor(t, []int{3, 1}, []float32{1, 1, 1})
		got, err := matMulRounded(a, b, false, false)
		if err != nil {
			t.Fatalf("matMulRounded: %v", err)
		}
		want := float32(1) + float32(math.Ldexp(1, -23))
		if got.data[0] != want {
			t.Fatalf("result = %v, want %v", got.data[0], want)
		}
	}
}

func TestMatMulRoundedRejectsInvalidOperands(t *testing.T) {
	valid := roundedTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6})
	rankOne := roundedTensor(t, []int{2}, []float32{1, 2})
	intTensor, err := NewInt64Tensor([]int{2, 3}, []int64{1, 2, 3, 4, 5, 6})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		a, b *Tensor
	}{
		{"nil left", nil, valid}, {"nil right", valid, nil}, {"incompatible inner dimensions", valid, roundedTensor(t, []int{4, 2}, make([]float32, 8))},
		{"rank one", rankOne, valid}, {"non float32", valid, intTensor},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := matMulRounded(tc.a, tc.b, false, false)
			if err == nil {
				t.Fatal("matMulRounded succeeded, want error")
			}
			if tc.name == "incompatible inner dimensions" || tc.name == "rank one" || tc.name == "non float32" {
				if !strings.Contains(err.Error(), "shape") {
					t.Errorf("error %q does not name the operand shapes", err)
				}
			}
		})
	}
}

func TestMatMulRoundedDoesNotModifyOperands(t *testing.T) {
	a, b := roundedTensor(t, []int{2, 2}, []float32{1, 2, 3, 4}), roundedTensor(t, []int{2, 2}, []float32{5, 6, 7, 8})
	aBefore, bBefore := append([]float32(nil), a.data...), append([]float32(nil), b.data...)
	if _, err := matMulRounded(a, b, false, false); err != nil {
		t.Fatalf("matMulRounded: %v", err)
	}
	if !reflect.DeepEqual(a.data, aBefore) || !reflect.DeepEqual(b.data, bBefore) {
		t.Fatal("matMulRounded modified an operand")
	}
}
