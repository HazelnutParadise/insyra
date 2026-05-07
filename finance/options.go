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

// Options controls precision and rounding of finance results.
// Zero-value Options selects high-precision defaults: scale=10, HalfUp.
type Options struct {
	// Scale is the number of decimal places kept in the result.
	// When 0, DefaultScale is used.
	Scale int32
	// Mode is the rounding mode applied to the result. Defaults to
	// RoundingModeHalfUp.
	Mode decimal.RoundingMode
}

// resolveOpts collapses a variadic Options slice into one effective set
// (last value wins) and fills in defaults.
func resolveOpts(opts []Options) Options {
	o := Options{Scale: DefaultScale, Mode: decimal.RoundingModeHalfUp}
	if len(opts) > 0 {
		last := opts[len(opts)-1]
		if last.Scale > 0 {
			o.Scale = last.Scale
		}
		o.Mode = last.Mode
	}
	return o
}

// outCtx returns the decimal.Context to apply to the final result.
func (o Options) outCtx() decimal.Context {
	return decimal.Context{Scale: o.Scale, Mode: o.Mode}
}

// workCtx returns the high-precision context used inside computations.
func (o Options) workCtx() decimal.Context {
	return decimal.Context{Scale: o.Scale + guardDigits, Mode: decimal.RoundingModeHalfUp}
}
