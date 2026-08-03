package dl

import (
	"math"
	"testing"
)

func TestF16BitsToFloat32Bits(t *testing.T) {
	cases := []struct {
		name string
		bits uint16
		want uint32
	}{
		{name: "positive zero", bits: 0x0000, want: 0x00000000},
		{name: "negative zero", bits: 0x8000, want: 0x80000000},
		{name: "one", bits: 0x3c00, want: 0x3f800000},
		{name: "negative two", bits: 0xc000, want: 0xc0000000},
		{name: "smallest subnormal", bits: 0x0001, want: 0x33800000},
		{name: "largest subnormal", bits: 0x03ff, want: 0x387fc000},
		{name: "positive infinity", bits: 0x7c00, want: 0x7f800000},
		{name: "negative infinity", bits: 0xfc00, want: 0xff800000},
		{name: "nan payload", bits: 0x7e01, want: 0x7fc02000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := math.Float32bits(f16BitsToFloat32(tc.bits)); got != tc.want {
				t.Fatalf("f16 bits %#04x widened to %#08x, want %#08x", tc.bits, got, tc.want)
			}
		})
	}
}

func TestBF16BitsToFloat32Bits(t *testing.T) {
	cases := []struct {
		name string
		bits uint16
		want uint32
	}{
		{name: "positive zero", bits: 0x0000, want: 0x00000000},
		{name: "negative zero", bits: 0x8000, want: 0x80000000},
		{name: "one", bits: 0x3f80, want: 0x3f800000},
		{name: "smallest subnormal", bits: 0x0001, want: 0x00010000},
		{name: "positive infinity", bits: 0x7f80, want: 0x7f800000},
		{name: "negative infinity", bits: 0xff80, want: 0xff800000},
		{name: "nan payload", bits: 0x7fc1, want: 0x7fc10000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := math.Float32bits(bf16BitsToFloat32(tc.bits)); got != tc.want {
				t.Fatalf("bf16 bits %#04x widened to %#08x, want %#08x", tc.bits, got, tc.want)
			}
		})
	}
}

func TestFloat32ToF16RoundToNearestEven(t *testing.T) {
	cases := []struct {
		name  string
		value float32
		want  uint16
	}{
		{name: "one", value: 1, want: 0x3c00},
		{name: "one point one", value: 1.1, want: 0x3c66},
		{name: "tie to even lower", value: math.Float32frombits(0x3f801000), want: 0x3c00},
		{name: "tie to even upper", value: math.Float32frombits(0x3f803000), want: 0x3c02},
		{name: "smallest subnormal tie to zero", value: math.Float32frombits(0x33000000), want: 0x0000},
		{name: "subnormal tie to even", value: math.Float32frombits(0x33800000 + 0x00400000), want: 0x0002},
		{name: "positive infinity", value: float32(math.Inf(1)), want: 0x7c00},
		{name: "negative infinity", value: float32(math.Inf(-1)), want: 0xfc00},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := float32ToF16Bits(tc.value); got != tc.want {
				t.Fatalf("f32 bits %#08x narrowed to %#04x, want %#04x", math.Float32bits(tc.value), got, tc.want)
			}
		})
	}
	if got := float32ToF16Bits(float32(math.NaN())); got&0x7c00 != 0x7c00 || got&0x03ff == 0 {
		t.Fatalf("f32 NaN narrowed to non-NaN f16 bits %#04x", got)
	}
}

func TestFloat32ToBF16RoundToNearestEven(t *testing.T) {
	cases := []struct {
		name  string
		value float32
		want  uint16
	}{
		{name: "one", value: 1, want: 0x3f80},
		{name: "tie to even lower", value: math.Float32frombits(0x3f808000), want: 0x3f80},
		{name: "tie to even upper", value: math.Float32frombits(0x3f818000), want: 0x3f82},
		{name: "positive infinity", value: float32(math.Inf(1)), want: 0x7f80},
		{name: "negative infinity", value: float32(math.Inf(-1)), want: 0xff80},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := float32ToBF16Bits(tc.value); got != tc.want {
				t.Fatalf("f32 bits %#08x narrowed to %#04x, want %#04x", math.Float32bits(tc.value), got, tc.want)
			}
		})
	}
	if got := float32ToBF16Bits(float32(math.NaN())); got&0x7f80 != 0x7f80 || got&0x007f == 0 {
		t.Fatalf("f32 NaN narrowed to non-NaN bf16 bits %#04x", got)
	}
}

func TestCastToHalfRoundsAndWidens(t *testing.T) {
	input, err := NewTensor([]int{4}, []float32{1.1, math.Float32frombits(0x3f801000), float32(math.Inf(1)), float32(math.NaN())})
	if err != nil {
		t.Fatalf("NewTensor: %v", err)
	}
	for _, dtype := range []DType{DTypeFloat16, DTypeBFloat16} {
		t.Run(string(dtype), func(t *testing.T) {
			got, err := Cast(input, dtype)
			if err != nil {
				t.Fatalf("Cast: %v", err)
			}
			if got.DType() != DTypeFloat32 {
				t.Fatalf("Cast dtype = %s, want %s", got.DType(), DTypeFloat32)
			}
			values := got.Data()
			for index, value := range input.Data() {
				if math.IsNaN(float64(value)) {
					if !math.IsNaN(float64(values[index])) {
						t.Fatalf("Cast NaN = %v", values[index])
					}
					continue
				}
				var want float32
				if dtype == DTypeFloat16 {
					want = f16BitsToFloat32(float32ToF16Bits(value))
				} else {
					want = bf16BitsToFloat32(float32ToBF16Bits(value))
				}
				if math.Float32bits(values[index]) != math.Float32bits(want) {
					t.Fatalf("Cast[%d] bits = %#08x, want %#08x", index, math.Float32bits(values[index]), math.Float32bits(want))
				}
			}
		})
	}
}
