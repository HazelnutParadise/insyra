package wgpu

import (
	"context"
	"math"
	"math/big"
	"math/rand"
	"os"
	"testing"
)

func requireResidentDevice(t *testing.T) {
	t.Helper()
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}
}

// roundedSum is the oracle: the exact sum of x*y terms rounded once to
// nearest-even float32, with an exactly-zero sum reported as +0. Operands are
// finite.
func roundedSum(terms [][2]float32) float32 {
	sum := new(big.Rat)
	for _, term := range terms {
		sum.Add(sum, new(big.Rat).Mul(new(big.Rat).SetFloat64(float64(term[0])), new(big.Rat).SetFloat64(float64(term[1]))))
	}
	if sum.Sign() == 0 {
		return 0
	}
	value, _ := new(big.Float).SetPrec(2048).SetRat(sum).Float32()
	return value
}

func sameBits(a, b float32) bool { return math.Float32bits(a) == math.Float32bits(b) }

func randomValues(r *rand.Rand, n int) []float32 {
	values := make([]float32, n)
	for i := range values {
		values[i] = float32(r.NormFloat64()) * float32(math.Ldexp(1, r.Intn(30)-15))
	}
	return values
}

func TestResidentBufferRoundTripKeepsEveryBit(t *testing.T) {
	requireResidentDevice(t)
	values := []float32{0, float32(math.Copysign(0, -1)), 1, -1.5, math.MaxFloat32, math.SmallestNonzeroFloat32,
		float32(math.Inf(1)), float32(math.Inf(-1)), math.Float32frombits(0x7fc00001), math.Float32frombits(0xffc12345)}
	buffer, err := Upload(values)
	if err != nil {
		t.Fatal(err)
	}
	defer buffer.Release()
	got, err := buffer.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for i := range values {
		if math.Float32bits(got[i]) != math.Float32bits(values[i]) {
			t.Fatalf("value %d: got bits %#x, want %#x", i, math.Float32bits(got[i]), math.Float32bits(values[i]))
		}
	}
}

func TestResidentEmptyBufferAndRelease(t *testing.T) {
	requireResidentDevice(t)
	buffer, err := Upload(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := buffer.Download(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("empty download = %v, %v", got, err)
	}
	buffer.Release()
	buffer.Release()
	if _, err := buffer.Download(context.Background()); err == nil {
		t.Fatal("download after release succeeded")
	}
}

func TestMatMulRoundedMatchesOracle(t *testing.T) {
	requireResidentDevice(t)
	r := rand.New(rand.NewSource(11))
	for _, shape := range [][3]int{{1, 1, 1}, {3, 5, 4}, {17, 130, 9}, {40, 3, 70}} {
		m, k, n := shape[0], shape[1], shape[2]
		for _, flags := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
			a, b := randomValues(r, m*k), randomValues(r, k*n)
			ab, err := Upload(a)
			if err != nil {
				t.Fatal(err)
			}
			bb, err := Upload(b)
			if err != nil {
				t.Fatal(err)
			}
			out, err := MatMulRounded(ab, bb, m, k, n, flags[0], flags[1])
			if err != nil {
				t.Fatal(err)
			}
			got, err := out.Download(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for row := 0; row < m; row++ {
				for col := 0; col < n; col++ {
					terms := make([][2]float32, k)
					for kk := 0; kk < k; kk++ {
						ai, bi := row*k+kk, kk*n+col
						if flags[0] {
							ai = kk*m + row
						}
						if flags[1] {
							bi = col*k + kk
						}
						terms[kk] = [2]float32{a[ai], b[bi]}
					}
					if want := roundedSum(terms); !sameBits(got[row*n+col], want) {
						t.Fatalf("[%d,%d,%d] trans=%v (%d,%d) = %g, want %g", m, k, n, flags, row, col, got[row*n+col], want)
					}
				}
			}
			ab.Release()
			bb.Release()
			out.Release()
		}
	}
}

func TestAddSumRowsAndSGDRoundedMatchOracle(t *testing.T) {
	requireResidentDevice(t)
	r := rand.New(rand.NewSource(5))
	m, n := 23, 11
	x, bias := randomValues(r, m*n), randomValues(r, n)
	xb, _ := Upload(x)
	bb, _ := Upload(bias)
	defer xb.Release()
	defer bb.Release()

	sum, err := AddRounded(xb, bb)
	if err != nil {
		t.Fatal(err)
	}
	defer sum.Release()
	gotSum, err := sum.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for i := range x {
		if want := roundedSum([][2]float32{{x[i], 1}, {bias[i%n], 1}}); !sameBits(gotSum[i], want) {
			t.Fatalf("add %d = %g, want %g", i, gotSum[i], want)
		}
	}

	rows, err := SumRowsRounded(xb, m, n)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Release()
	gotRows, err := rows.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for col := 0; col < n; col++ {
		terms := make([][2]float32, m)
		for row := 0; row < m; row++ {
			terms[row] = [2]float32{x[row*n+col], 1}
		}
		if want := roundedSum(terms); !sameBits(gotRows[col], want) {
			t.Fatalf("sum rows %d = %g, want %g", col, gotRows[col], want)
		}
	}

	w, g := randomValues(r, 500), randomValues(r, 500)
	wb, _ := Upload(w)
	gb, _ := Upload(g)
	defer wb.Release()
	defer gb.Release()
	rate := float32(0.0123)
	if err := SGDRounded(wb, gb, math.Float32bits(rate)^0x80000000); err != nil {
		t.Fatal(err)
	}
	gotW, err := wb.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for i := range w {
		if want := roundedSum([][2]float32{{w[i], 1}, {-rate, g[i]}}); !sameBits(gotW[i], want) {
			t.Fatalf("sgd %d = %g, want %g", i, gotW[i], want)
		}
	}
}

func TestManyDispatchesWithoutReadbackStayBounded(t *testing.T) {
	requireResidentDevice(t)
	x, _ := Upload([]float32{1, 2, 3, 4})
	defer x.Release()
	one, _ := Upload([]float32{1, 1, 1, 1})
	defer one.Release()
	for i := 0; i < 2000; i++ {
		out, err := AddRounded(x, one)
		if err != nil {
			t.Fatalf("dispatch %d: %v", i, err)
		}
		out.Release()
	}
	if err := Synchronize(); err != nil {
		t.Fatal(err)
	}
}
