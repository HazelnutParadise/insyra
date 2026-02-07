package utils

import (
	"fmt"
	"testing"
)

func TestConvertDateFormat(t *testing.T) {
	formatStr := "YYYY-MM-DD"
	goFormat := ConvertDateFormat(formatStr)
	if goFormat != "2006-01-02" {
		t.Errorf("Expected 2006-01-02, got %s", goFormat)
	}

	formatStr = "YYYY-MM-DD HH:mm:ss"
	goFormat = ConvertDateFormat(formatStr)
	if goFormat != "2006-01-02 15:04:05" {
		t.Errorf("Expected 2006-01-02 15:04:05, got %s", goFormat)
	}
}

func TestFormatValueArrays(t *testing.T) {
	// 測試空陣列
	if result := FormatValue([]int{}); result != "[]" {
		t.Errorf("Expected [], got %s", result)
	}

	// 測試長度為1的陣列
	if result := FormatValue([]int{1}); result != "[1]" {
		t.Errorf("Expected [1], got %s", result)
	}

	// 測試長度為2的陣列
	if result := FormatValue([]int{1, 2}); result != "[1, 2]" {
		t.Errorf("Expected [1, 2], got %s", result)
	}

	// 測試長度為3的陣列
	if result := FormatValue([]int{1, 2, 3}); result != "[1, 2, 3]" {
		t.Errorf("Expected [1, 2, 3], got %s", result)
	}

	// 測試長度大於3的陣列
	if result := FormatValue([]int{1, 2, 3, 4, 5}); result != "[1, 2, ... +3]" {
		t.Errorf("Expected [1, 2, ... +3], got %s", result)
	}

	// 測試固定大小陣列
	var arr = [3]int{1, 2, 3}
	if result := FormatValue(arr); result != "[1, 2, 3]" {
		t.Errorf("Expected [1, 2, 3], got %s", result)
	}

	// 測試字符串陣列
	if result := FormatValue([]string{"a", "b"}); result != "[a, b]" {
		t.Errorf("Expected [a, b], got %s", result)
	}
}

func TestCalcParseColIndexRoundtrip(t *testing.T) {
	cases := []int{0, 1, 25, 26, 27, 51, 52, 701, 702, 16383, 16384, 100000}
	for _, n := range cases {
		s, ok := CalcColIndex(n)
		if !ok {
			t.Fatalf("CalcColIndex(%d) returned not ok", n)
		}
		m, ok2 := ParseColIndex(s)
		if !ok2 || m != n {
			t.Errorf("roundtrip failed for %d -> %s -> %d (ok2=%v)", n, s, m, ok2)
		}
	}

	// negative case
	s, ok := CalcColIndex(-1)
	if ok || s != "" {
		t.Errorf("expected invalid for -1, got %v %v", s, ok)
	}
}

func BenchmarkCalcColIndex(b *testing.B) {
	b.ReportAllocs()
	ns := []int{0, 1, 25, 26, 701, 702, 1000, 10000, 100000}
	for _, n := range ns {
		b.Run(fmt.Sprintf("n_%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = CalcColIndex(n)
			}
		})
	}
}

func TestParseColIndex(t *testing.T) {
	cases := []struct {
		s    string
		want int
		ok   bool
	}{
		{"A", 0, true},
		{"Z", 25, true},
		{"AA", 26, true},
		{"ZZ", 701, true},
		{"a", 0, true},
		{"zz", 701, true},
		{"", -1, false},
		{"A1", -1, false},
		{"@", -1, false},
	}
	for _, c := range cases {
		got, ok := ParseColIndex(c.s)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseColIndex(%q) = %d, %v; want %d, %v", c.s, got, ok, c.want, c.ok)
		}
	}
}

func BenchmarkParseColIndex(b *testing.B) {
	ns := []string{"A", "Z", "AA", "ZZ", "AAA", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", "ZZZZZZ"}
	b.ReportAllocs()
	for _, s := range ns {
		b.Run(s, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = ParseColIndex(s)
			}
		})
	}
}
