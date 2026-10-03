package ccl

import (
	"fmt"
	"math"
)

// Integers stay integers. Two operands of Go integer types meet in int64, so
// int64(9007199254740993) + 0 keeps its last digit and an integer column comes
// out of A * 1 as int64. A result int64 cannot hold is an error rather than a
// wrapped number or a float64 that quietly lost digits. Division, powers and
// anything involving a float64, a numeric string or a bool go through float64
// as before.

// exactInt reads v as an int64 when it is a Go integer whose value fits one. A
// time.Duration is a date difference, counted in seconds elsewhere, not an
// integer, and a uint64 past the int64 range is left to the float64 path.
func exactInt(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int8:
		return int64(x), true
	case int16:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case uint:
		if uint64(x) <= math.MaxInt64 {
			return int64(x), true
		}
	case uint8:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint64:
		if x <= math.MaxInt64 {
			return int64(x), true
		}
	}
	return 0, false
}

// applyIntegerOperator applies op when both operands are integers. ok is false
// when op is not one integers keep, or the operands are not both integers; the
// caller then goes on to its other rules.
//
// In arithmetic nil counts as 0, as it always has, and next to an integer it
// is an integer 0. A missing cell carries no type, and this keeps an integer
// column with missing cells an integer column through A + 1, so its values
// stay exact when it is written out. A comparison with nil keeps its own rule.
func applyIntegerOperator(op string, left, right any) (res any, ok bool, err error) {
	a, aok := exactInt(left)
	b, bok := exactInt(right)
	switch op {
	case "+", "-", "*", "%":
		integers := (aok && bok) || (aok && right == nil) || (bok && left == nil)
		if !integers {
			return nil, false, nil
		}
		r, err := integerArithmetic(op, a, b)
		return r, true, err
	case "==", "!=", "<", ">", "<=", ">=":
		if !aok || !bok {
			return nil, false, nil
		}
		switch op {
		case "==":
			return a == b, true, nil
		case "!=":
			return a != b, true, nil
		case "<":
			return a < b, true, nil
		case ">":
			return a > b, true, nil
		case "<=":
			return a <= b, true, nil
		default:
			return a >= b, true, nil
		}
	}
	return nil, false, nil
}

// integerArithmetic computes a op b for +, -, * and %, refusing a result
// outside int64.
func integerArithmetic(op string, a, b int64) (any, error) {
	switch op {
	case "+":
		if r, ok := addInt64(a, b); ok {
			return r, nil
		}
	case "-":
		if (b < 0 && a > math.MaxInt64+b) || (b > 0 && a < math.MinInt64+b) {
			break
		}
		return a - b, nil
	case "*":
		if r, ok := mulInt64(a, b); ok {
			return r, nil
		}
	case "%":
		if b == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		// Go defines MinInt64 % -1 as 0, so % cannot overflow.
		return a % b, nil
	}
	return nil, fmt.Errorf("integer overflow: %d %s %d does not fit in an int64; write one operand as a decimal, such as %d.0, to compute in float64", a, op, b, a)
}

func addInt64(a, b int64) (int64, bool) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, false
	}
	return a + b, true
}

func mulInt64(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	r := a * b
	if r/b != a || (a == -1 && b == math.MinInt64) || (b == -1 && a == math.MinInt64) {
		return 0, false
	}
	return r, true
}

// integerAggregate tracks, next to an aggregate's float64 arithmetic, whether
// every value it used was an integer, and if so the exact int64 answer. SUM,
// MIN and MAX, and their streaming forms, share it so the two cannot answer
// differently.
type integerAggregate struct {
	kind     string // SUM, MIN or MAX
	value    int64
	used     bool // at least one value
	allInt   bool // every value used was an integer
	overflow bool // the int64 sum left the range
}

func newIntegerAggregate(kind string) integerAggregate {
	return integerAggregate{kind: kind, allInt: true}
}

// add records v, a value the aggregate used in its float64 arithmetic.
func (g *integerAggregate) add(v any) {
	if !g.allInt {
		return // the float64 answer stands; nothing here can change that
	}
	i, ok := exactInt(v)
	if !ok {
		g.allInt = false
		g.used = true
		return
	}
	if !g.used {
		g.value = i
		g.used = true
		return
	}
	switch g.kind {
	case "SUM":
		if g.overflow {
			return
		}
		if r, ok := addInt64(g.value, i); ok {
			g.value = r
		} else {
			g.overflow = true
		}
	case "MIN":
		g.value = min(g.value, i)
	case "MAX":
		g.value = max(g.value, i)
	}
}

// result is the int64 answer when every value used was an integer; ok is false
// otherwise, and the float64 answer stands.
func (g *integerAggregate) result() (res any, ok bool, err error) {
	if !g.used || !g.allInt {
		return nil, false, nil
	}
	if g.overflow {
		return nil, true, fmt.Errorf("integer overflow: %s of these integers does not fit in an int64", g.kind)
	}
	return g.value, true, nil
}
