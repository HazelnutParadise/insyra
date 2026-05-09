package finance

import "github.com/TimLai666/go-decimal/decimal"

// PaymentTiming indicates whether an annuity payment occurs at the end
// of each period (ordinary annuity, Excel's type=0) or at the beginning
// (annuity due, Excel's type=1).
type PaymentTiming uint8

const (
	// PaymentEnd places each payment at the end of its period.
	// Equivalent to Excel's type=0. This is the default.
	PaymentEnd PaymentTiming = 0
	// PaymentBegin places each payment at the start of its period.
	// Equivalent to Excel's type=1.
	PaymentBegin PaymentTiming = 1
)

// DefaultScale is the result scale used when Options.Scale == 0.
const DefaultScale int32 = 10

// guardDigits are added to Scale when forming the internal working
// context, so chained operations don't lose accuracy before the final
// normalize back to the requested output scale.
const guardDigits int32 = 16

// RoundingMode is a string-typed selector for the rounding strategy
// applied to finance results. A string enum is used (rather than the
// underlying decimal.RoundingMode uint8) so that the empty string is
// unambiguously "use default" — there is no zero-value collision with
// any real mode.
type RoundingMode string

const (
	// RoundHalfUp is the everyday 四捨五入 rule: half values move away
	// from zero. 1.235 → 1.24, -1.235 → -1.24. This is the default.
	RoundHalfUp RoundingMode = "half-up"
	// RoundHalfEven is banker's rounding (IEEE 754 / Python decimal
	// default): half values pick the neighbor whose last digit is
	// even, eliminating HalfUp's systematic upward bias on long sums.
	// 1.225 → 1.22, 1.235 → 1.24.
	RoundHalfEven RoundingMode = "half-even"
	// RoundHalfDown is "round half toward zero": half values stay put
	// rather than stepping away. 1.235 → 1.23, -1.235 → -1.23.
	RoundHalfDown RoundingMode = "half-down"
	// RoundUp rounds away from zero on any non-zero residue.
	// 1.231 → 1.24, -1.231 → -1.24.
	RoundUp RoundingMode = "up"
	// RoundDown truncates toward zero (the absolute value never grows).
	// 1.235 → 1.23, -1.235 → -1.23.
	RoundDown RoundingMode = "down"
	// RoundCeiling rounds toward +∞ on any non-zero residue.
	// 1.231 → 1.24, -1.231 → -1.23.
	RoundCeiling RoundingMode = "ceiling"
	// RoundFloor rounds toward -∞ on any non-zero residue.
	// 1.231 → 1.23, -1.231 → -1.24.
	RoundFloor RoundingMode = "floor"
	// Round05Up implements Python's ROUND_05UP rule: after truncating
	// toward zero, if the kept last digit is 0 or 5 then any non-zero
	// residue causes a step away from zero; otherwise the residue is
	// dropped.
	Round05Up RoundingMode = "05up"
	// RoundUnnecessary asserts that no rounding will be required. If
	// an operation under this mode has to discard a non-zero residue,
	// it panics with decimal.ErrRoundingNecessary.
	RoundUnnecessary RoundingMode = "unnecessary"
)

// toDecimal maps a finance RoundingMode to the underlying
// decimal.RoundingMode. An empty string or any unrecognized value
// falls through to the package default (HalfUp), so callers cannot
// silently pick up an unintended mode by leaving the field zero.
func (m RoundingMode) toDecimal() decimal.RoundingMode {
	switch m {
	case RoundHalfEven:
		return decimal.RoundingModeHalfEven
	case RoundHalfDown:
		return decimal.RoundingModeHalfDown
	case RoundUp:
		return decimal.RoundingModeUp
	case RoundDown:
		return decimal.RoundingModeDown
	case RoundCeiling:
		return decimal.RoundingModeCeiling
	case RoundFloor:
		return decimal.RoundingModeFloor
	case Round05Up:
		return decimal.RoundingMode05Up
	case RoundUnnecessary:
		return decimal.RoundingModeUnnecessary
	}
	return decimal.RoundingModeHalfUp
}

// Options controls precision and rounding of finance results.
// Zero-value Options selects the high-precision defaults: Scale=10 and
// RoundHalfUp. Scale and Mode resolve independently — leaving either
// at its zero value (0 / "") keeps that field's default while letting
// the other be customized.
type Options struct {
	// Scale is the number of decimal places kept in the result.
	// When 0, DefaultScale (10) is used.
	Scale int32
	// Mode is the rounding mode applied to the result. When empty,
	// RoundHalfUp is used. Available values: RoundHalfUp, RoundHalfEven,
	// RoundHalfDown, RoundUp, RoundDown, RoundCeiling, RoundFloor,
	// Round05Up, RoundUnnecessary.
	Mode RoundingMode
}

// resolveOpts collapses a variadic Options slice into one effective set
// (last value wins) and fills in defaults. Each field's zero value is
// treated as "use default", so partial Options literals work as expected.
func resolveOpts(opts []Options) Options {
	o := Options{Scale: DefaultScale, Mode: RoundHalfUp}
	if len(opts) > 0 {
		last := opts[len(opts)-1]
		if last.Scale > 0 {
			o.Scale = last.Scale
		}
		if last.Mode != "" {
			o.Mode = last.Mode
		}
	}
	return o
}

// outCtx returns the decimal.Context to apply to the final result.
func (o Options) outCtx() decimal.Context {
	return decimal.Context{Scale: o.Scale, Mode: o.Mode.toDecimal()}
}

// workCtx returns the high-precision context used inside computations.
func (o Options) workCtx() decimal.Context {
	return decimal.Context{Scale: o.Scale + guardDigits, Mode: decimal.RoundingModeHalfUp}
}
