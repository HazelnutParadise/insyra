package ccl

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// roundWritten rounds f half away from zero to digits decimal places, reading f
// as its shortest decimal text. Multiplying instead would see the binary value:
// the float64 closest to 1.005 lies just below it, and 1.005 would round to 1.
func roundWritten(f float64, digits int) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return f
	}
	text := strconv.FormatFloat(math.Abs(f), 'f', -1, 64)
	whole, frac, _ := strings.Cut(text, ".")
	all := whole + frac
	keep := len(whole) + digits
	if keep >= len(all) {
		return f
	}
	if keep < 0 {
		return 0
	}
	kept := []byte(all[:keep])
	if all[keep] >= '5' {
		i := len(kept) - 1
		for i >= 0 && kept[i] == '9' {
			kept[i] = '0'
			i--
		}
		if i >= 0 {
			kept[i]++
		} else {
			kept = append([]byte{'1'}, kept...)
		}
	}
	// The last kept digit stands digits places after the point.
	result, err := strconv.ParseFloat("0"+string(kept)+"e"+strconv.Itoa(-digits), 64)
	if err != nil {
		return f
	}
	if result == 0 {
		return 0
	}
	if f < 0 {
		return -result
	}
	return result
}

// registerMathFunctions registers Excel/SQL-style scalar math functions.
// All functions accept any value coercible to float64 via toFloat64; on
// failure they return a typed error so the evaluator can surface it.
func registerMathFunctions() {
	registerFunction("ABS", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("ABS requires 1 argument")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("ABS: cannot convert %T to number", args[0])
		}
		return math.Abs(f), nil
	})

	registerFunction("ROUND", func(args ...any) (any, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("ROUND requires 1 or 2 arguments")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("ROUND: cannot convert %T to number", args[0])
		}
		digits := 0
		if len(args) == 2 {
			d, ok := toFloat64(args[1])
			if !ok {
				return nil, fmt.Errorf("ROUND: digits arg must be a number, got %T", args[1])
			}
			n, err := clampedInt(d, "ROUND: digits")
			if err != nil {
				return nil, err
			}
			digits = n
		}
		return roundWritten(f, digits), nil
	})

	registerFunction("FLOOR", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("FLOOR requires 1 argument")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("FLOOR: cannot convert %T to number", args[0])
		}
		return math.Floor(f), nil
	})

	registerFunction("CEIL", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("CEIL requires 1 argument")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("CEIL: cannot convert %T to number", args[0])
		}
		return math.Ceil(f), nil
	})

	registerFunction("TRUNC", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("TRUNC requires 1 argument")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("TRUNC: cannot convert %T to number", args[0])
		}
		return math.Trunc(f), nil
	})

	registerFunction("MOD", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("MOD requires 2 arguments")
		}
		a, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("MOD: cannot convert %T to number", args[0])
		}
		b, ok := toFloat64(args[1])
		if !ok {
			return nil, fmt.Errorf("MOD: cannot convert %T to number", args[1])
		}
		if b == 0 {
			return nil, fmt.Errorf("MOD: division by zero")
		}
		return modulo(a, b), nil
	})

	registerFunction("POW", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("POW requires 2 arguments")
		}
		base, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("POW: cannot convert %T to number", args[0])
		}
		exp, ok := toFloat64(args[1])
		if !ok {
			return nil, fmt.Errorf("POW: cannot convert %T to number", args[1])
		}
		return math.Pow(base, exp), nil
	})

	registerFunction("SQRT", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("SQRT requires 1 argument")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("SQRT: cannot convert %T to number", args[0])
		}
		if f < 0 {
			return nil, fmt.Errorf("SQRT: cannot take square root of negative number %v", f)
		}
		return math.Sqrt(f), nil
	})

	registerFunction("LN", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("LN requires 1 argument")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("LN: cannot convert %T to number", args[0])
		}
		if f <= 0 {
			return nil, fmt.Errorf("LN: cannot take natural log of non-positive number %v", f)
		}
		return math.Log(f), nil
	})

	// LOG matches Excel: LOG(x) defaults to base 10; LOG(x, base) uses given base.
	registerFunction("LOG", func(args ...any) (any, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("LOG requires 1 or 2 arguments")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("LOG: cannot convert %T to number", args[0])
		}
		if f <= 0 {
			return nil, fmt.Errorf("LOG: cannot take log of non-positive number %v", f)
		}
		if len(args) == 1 {
			return math.Log10(f), nil
		}
		base, ok := toFloat64(args[1])
		if !ok {
			return nil, fmt.Errorf("LOG: cannot convert base %T to number", args[1])
		}
		if base <= 0 || base == 1 {
			return nil, fmt.Errorf("LOG: invalid base %v", base)
		}
		return math.Log(f) / math.Log(base), nil
	})

	registerFunction("LOG10", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("LOG10 requires 1 argument")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("LOG10: cannot convert %T to number", args[0])
		}
		if f <= 0 {
			return nil, fmt.Errorf("LOG10: cannot take log of non-positive number %v", f)
		}
		return math.Log10(f), nil
	})

	registerFunction("EXP", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("EXP requires 1 argument")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("EXP: cannot convert %T to number", args[0])
		}
		return math.Exp(f), nil
	})

	registerFunction("SIGN", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("SIGN requires 1 argument")
		}
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("SIGN: cannot convert %T to number", args[0])
		}
		switch {
		case f > 0:
			return 1.0, nil
		case f < 0:
			return -1.0, nil
		default:
			return 0.0, nil
		}
	})
}
