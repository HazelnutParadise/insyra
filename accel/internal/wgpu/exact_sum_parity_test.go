package wgpu

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"testing"
)

// exactSumCategoryTotal is the per-category size of a row set: how many rows
// carry that name, and how many operand pairs those rows hold between them.
type exactSumCategoryTotal struct {
	name  string
	rows  int
	pairs int
}

// exactSumParityRows builds the adversarial row set TestExactSumDeviceMatchesCPU
// checks. Every row is a list of float32 bit-pattern pairs whose exact product
// sum the library has to reproduce, and names[i] is the category of rows[i].
// Categories are emitted in the order below, and every draw comes from r, so a
// given seed always yields the same set.
//
// The categories are chosen so each one attacks a different part of the
// register: "moderate" is ordinary data, "bits" is every float32 pattern
// including NaN, infinity and subnormals, "cancel" pairs every term with its
// negation and leaves one small value, "tie" lands exactly half-way between two
// float32 values, "subnormal" keeps one operand below the smallest normal,
// "overflow" pushes a largest finite value past the point where it must become
// infinite, "long" forces hundreds of carry passes, and "ripple" adds powers of
// two that carry across the whole register.
func exactSumParityRows(r *rand.Rand) (names []string, rows [][][2]uint32) {
	ld := func(v float64, e int) float32 { return float32(math.Ldexp(v, e)) }
	bits := func(v float32) uint32 { return math.Float32bits(v) }
	one := bits(float32(1))

	// moderate returns a pair whose operands are normal float32 values near 1
	// scaled by a power of two in [-60, 60], the shape ordinary data has.
	moderate := func() [2]uint32 {
		return [2]uint32{
			bits(ld(r.NormFloat64(), r.Intn(121)-60)),
			bits(ld(r.NormFloat64(), r.Intn(121)-60)),
		}
	}

	// signOf returns a random sign for v, and is the only place a sign is
	// drawn, so the sign pattern of a row is visible in the generator.
	signOf := func(v float32) float32 {
		if r.Intn(2) == 0 {
			return v
		}
		return -v
	}

	add := func(name string, row [][2]uint32) {
		names = append(names, name)
		rows = append(rows, row)
	}

	for n := 0; n < 1000; n++ {
		row := make([][2]uint32, r.Intn(301))
		for i := range row {
			row[i] = moderate()
		}
		add("moderate", row)
	}

	for n := 0; n < 1000; n++ {
		row := make([][2]uint32, 1+r.Intn(50))
		for i := range row {
			row[i] = [2]uint32{r.Uint32(), r.Uint32()}
		}
		add("bits", row)
	}

	for n := 0; n < 300; n++ {
		pairs := 1 + r.Intn(40)
		row := make([][2]uint32, 0, 2*pairs+1)
		for i := 0; i < pairs; i++ {
			pair := moderate()
			row = append(row, pair, [2]uint32{pair[0] ^ 0x80000000, pair[1]})
		}
		row = append(row, [2]uint32{bits(ld(1, r.Intn(250)-149)), one})
		r.Shuffle(len(row), func(i, j int) { row[i], row[j] = row[j], row[i] })
		add("cancel", row)
	}

	for n := 0; n < 300; n++ {
		f := ld(1+r.Float64(), r.Intn(200)-100)
		h := (math.Nextafter32(f, float32(math.Inf(1))) - f) * 0.5
		row := [][2]uint32{{bits(f), one}, {bits(h), one}}
		if n%2 == 1 {
			row = append(row, [2]uint32{bits(ld(1, r.Intn(20)-170)), one})
		}
		add("tie", row)
	}

	for n := 0; n < 300; n++ {
		row := make([][2]uint32, 1+r.Intn(20))
		for i := range row {
			x := signOf(math.Float32frombits(uint32(r.Intn(1 << 23))))
			y := signOf(float32(0.5 + 1.5*r.Float64()))
			row[i] = [2]uint32{bits(x), bits(y)}
		}
		add("subnormal", row)
	}

	for n := 0; n < 200; n++ {
		row := [][2]uint32{
			{bits(math.MaxFloat32), one},
			{bits(ld(1, 100+r.Intn(6))), bits(signOf(float32(1)))},
		}
		if n%2 == 1 {
			row = append(row, [2]uint32{bits(ld(1, r.Intn(40)-20)), one})
		}
		add("overflow", row)
	}

	for n := 0; n < 20; n++ {
		row := make([][2]uint32, 5000)
		for i := range row {
			row[i] = [2]uint32{
				bits(ld(r.NormFloat64(), r.Intn(121)-60)),
				bits(signOf(ld(r.NormFloat64(), r.Intn(121)-60))),
			}
		}
		add("long", row)
	}

	for n := 0; n < 300; n++ {
		row := make([][2]uint32, 2+r.Intn(5))
		for i := range row {
			a := r.Intn(250) - 149
			b := r.Intn(250) - 149
			// The sign lands on x only: with the same sign on both operands
			// every product is positive, so the sum never borrows and the
			// carry chain is never walked backwards.
			row[i] = [2]uint32{bits(ld(float64(signOf(float32(1))), a)), bits(ld(1, b))}
		}
		add("ripple", row)
	}

	return names, rows
}

// exactSumCategoryTotals counts rows and operand pairs per category, in the
// order each category first appears in names.
func exactSumCategoryTotals(names []string, rows [][][2]uint32) []exactSumCategoryTotal {
	totals := make([]exactSumCategoryTotal, 0, 8)
	index := make(map[string]int, 8)
	for i, name := range names {
		at, ok := index[name]
		if !ok {
			at = len(totals)
			index[name] = at
			totals = append(totals, exactSumCategoryTotal{name: name})
		}
		totals[at].rows++
		totals[at].pairs += len(rows[i])
	}
	return totals
}

// exactSumPairPrefix renders up to the first four operand pairs of a row as
// float32 bit patterns. A mismatch cannot be reproduced without them, and the
// rest of the row is not printed so one row cannot flood the log.
func exactSumPairPrefix(row [][2]uint32) string {
	limit := len(row)
	if limit > 4 {
		limit = 4
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, fmt.Sprintf("(%#08x,%#08x)", row[i][0], row[i][1]))
	}
	return strings.Join(parts, " ")
}

// TestExactSumDeviceMatchesCPU sums every row of exactSumParityRows on the
// device and holds the result to the exact math/big oracle, both for the
// single-register accumulation and for the two-register split, so a row whose
// exact sum the device gets wrong is reported with the operand bits needed to
// reproduce it.
func TestExactSumDeviceMatchesCPU(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	names, rows := exactSumParityRows(rand.New(rand.NewSource(35)))
	if len(names) != len(rows) {
		t.Fatalf("row set has %d names for %d rows", len(names), len(rows))
	}

	single, split, err := runExactSumHarness(context.Background(), rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(single) != len(rows) || len(split) != len(rows) {
		t.Fatalf("harness returned %d single and %d split outputs for %d rows", len(single), len(split), len(rows))
	}

	totals := exactSumCategoryTotals(names, rows)
	wrong := make(map[string]int, len(totals))
	printed := make(map[string]int, len(totals))
	for i := range rows {
		oracle := exactSumOracleBits(rows[i])
		if single[i] == oracle && split[i] == oracle {
			continue
		}
		name := names[i]
		wrong[name]++
		if printed[name] == 20 {
			continue
		}
		printed[name]++
		t.Errorf("%s row %d of %d: %d pairs, oracle=%#08x single=%#08x split=%#08x, first pairs %s",
			name, i, len(rows), len(rows[i]), oracle, single[i], split[i], exactSumPairPrefix(rows[i]))
	}

	for _, total := range totals {
		if n := wrong[total.name]; n > 0 {
			t.Errorf("%s: %d of %d rows disagree with the exact oracle", total.name, n, total.rows)
		}
	}
	for _, total := range totals {
		t.Logf("%s: %d rows, %d pairs, %d mismatching rows", total.name, total.rows, total.pairs, wrong[total.name])
	}
}
