package ccl

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ClockNow is what NOW reads: the local clock's date and time, with no zone. A
// date CCL reads from text without a zone is held as UTC, so the local clock is
// held the same way, and DATEDIFF(TODAY(), '2024-01-02', 'day') is a whole number
// wherever the program runs.
func ClockNow() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)
}

func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// FreezeClock replaces every TODAY() and NOW() in a compiled expression with
// the date and time given, so every row of one call reads the same moment.
func FreezeClock(n CCLNode, now time.Time) CCLNode {
	switch t := n.(type) {
	case *funcCallNode:
		if len(t.args) == 0 {
			switch strings.ToUpper(t.name) {
			case "NOW":
				return literalNode(now)
			case "TODAY":
				return literalNode(dateOf(now))
			}
		}
		args := make([]cclNode, len(t.args))
		for i, arg := range t.args {
			args[i] = FreezeClock(arg, now)
		}
		return &funcCallNode{name: t.name, args: args}
	case *cclBinaryOpNode:
		return &cclBinaryOpNode{op: t.op, left: FreezeClock(t.left, now), right: FreezeClock(t.right, now)}
	case *cclFoldChainNode:
		operands := make([]cclNode, len(t.operands))
		for i, operand := range t.operands {
			operands[i] = FreezeClock(operand, now)
		}
		return &cclFoldChainNode{init: FreezeClock(t.init, now), ops: t.ops, operands: operands}
	case *cclChainedComparisonNode:
		values := make([]cclNode, len(t.values))
		for i, v := range t.values {
			values[i] = FreezeClock(v, now)
		}
		return &cclChainedComparisonNode{ops: t.ops, values: values}
	case *cclAssignmentNode:
		return &cclAssignmentNode{target: t.target, expr: FreezeClock(t.expr, now)}
	case *cclNewColNode:
		return &cclNewColNode{colName: t.colName, expr: FreezeClock(t.expr, now)}
	}
	return n
}

// dateArgument reads one number of DATE, dropping any fraction as Excel does.
func dateArgument(name string, v any) (int, error) {
	switch v.(type) {
	case bool, time.Time, time.Duration:
		return 0, fmt.Errorf("DATE: %s is %v (%T), which is not a number", name, v, v)
	}
	f, ok := toFloat64(v)
	if !ok || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("DATE: %s is %v (%T), which is not a number", name, v, v)
	}
	if math.Abs(f) > 1e7 {
		return 0, fmt.Errorf("DATE: %s %v is out of range", name, v)
	}
	return int(f), nil
}

func registerClockFunctions() {
	registerFunction("TODAY", func(args ...any) (any, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("TODAY takes no arguments")
		}
		return dateOf(ClockNow()), nil
	})
	registerFunction("NOW", func(args ...any) (any, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("NOW takes no arguments")
		}
		return ClockNow(), nil
	})
	// DATE carries over the way Excel's does: month 13 is January of the next
	// year, and day 0 is the last day of the month before.
	registerFunction("DATE", func(args ...any) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("DATE requires 3 arguments: year, month, day")
		}
		if args[0] == nil || args[1] == nil || args[2] == nil {
			return nil, nil
		}
		year, err := dateArgument("year", args[0])
		if err != nil {
			return nil, err
		}
		month, err := dateArgument("month", args[1])
		if err != nil {
			return nil, err
		}
		day, err := dateArgument("day", args[2])
		if err != nil {
			return nil, err
		}
		d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if d.Year() < 1 || d.Year() > 9999 {
			return nil, fmt.Errorf("DATE(%d, %d, %d) is outside the years 1 to 9999", year, month, day)
		}
		return d, nil
	})
}
