package utils

import (
	"fmt"
	"math"
	"strconv"
)

// FloatText writes a float as text by the rule encoding/json uses: a plain
// decimal for magnitudes from 1e-6 up to (not including) 1e21, and Go's
// shortest exponent form outside that range. NaN and ±Inf come out as "NaN",
// "+Inf" and "-Inf".
//
// fmt's %v switches to an exponent as soon as a value reaches one million, so
// a revenue of 1,500,000 used to come out as "1.5e+06" in CCL strings and CSV
// cells while ToJSON, which goes through encoding/json, wrote 1500000. A value
// that stays in exponent form here is written exactly as %v wrote it.
func FloatText(f float64, bitSize int) string {
	if abs := math.Abs(f); abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		return strconv.FormatFloat(f, 'e', -1, bitSize)
	}
	return strconv.FormatFloat(f, 'f', -1, bitSize)
}

// ValueText writes a cell value as text for output. Floats follow FloatText;
// every other type keeps the text fmt.Sprint gives it, nil included —
// OneHotEncode and Pivot name a nil category "<nil>" on purpose.
func ValueText(v any) string {
	switch x := v.(type) {
	case float64:
		return FloatText(x, 64)
	case float32:
		return FloatText(float64(x), 32)
	case string:
		return x
	}
	return fmt.Sprint(v)
}
