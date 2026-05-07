package finance

import (
	"errors"
	"math/big"

	"github.com/TimLai666/go-decimal/decimal"
)

// internalCtx is used by New / MustNew when callers haven't picked a
// context yet. 28 digits is enough headroom for chained TVM operations
// while keeping parsing fast.
var internalCtx = decimal.Context{Scale: 28, Mode: decimal.RoundingModeHalfUp}

// oneBigInt is reused by helpers that need the literal big.Int(1) —
// notably tenToMinus, which builds 10^-k by treating 1 as a scaled int.
var oneBigInt = big.NewInt(1)

// Zero is a convenience zero-valued decimal at the package's internal
// precision. Pass it as fv (or pv) when a TVM call doesn't need that
// term.
var Zero = decimal.NewFromInt64(internalCtx, 0)

// New parses s as a decimal at the package's internal precision (28
// digits). Use this to build inputs for finance functions from
// human-readable strings.
//
//	rate := finance.MustNew("0.00375")
//	pv   := finance.MustNew("250000")
func New(s string) (decimal.Decimal, error) {
	return decimal.Parse(internalCtx, s)
}

// MustNew is like New but panics on parse error. Suitable for tests and
// constants.
func MustNew(s string) decimal.Decimal {
	return decimal.MustParse(internalCtx, s)
}

// FromInt converts an int to a Decimal at the package's internal
// precision.
func FromInt(n int) decimal.Decimal {
	return decimal.NewFromInt64(internalCtx, int64(n))
}

// FromFloat converts a float64 to a Decimal. Use sparingly: float
// inputs already carry binary rounding error before they reach the
// decimal layer.
func FromFloat(f float64) (decimal.Decimal, error) {
	return decimal.Parse(internalCtx, formatFloat(f))
}

// formatFloat renders f at full precision so Parse can capture every
// representable digit; we don't try to "clean it up" since the caller
// chose to start from float.
func formatFloat(f float64) string {
	return big.NewFloat(f).Text('f', -1)
}

// isZero reports whether d compares equal to 0.
func isZero(d decimal.Decimal) bool {
	return decimal.Cmp(d, Zero) == 0
}

// neg returns -d at the working context.
func neg(d decimal.Decimal) decimal.Decimal {
	return decimal.Neg(d)
}

// onePlus returns 1 + r in the given context.
func onePlus(ctx decimal.Context, r decimal.Decimal) decimal.Decimal {
	return decimal.Add(ctx, decimal.NewFromInt64(ctx, 1), r)
}

// powInt raises base to the integer power n in the given context.
// Uses Pow under the hood; Pow detects integer exponents and uses
// square-and-multiply (exact, no Log/Exp involved).
func powInt(ctx decimal.Context, base decimal.Decimal, n int) (decimal.Decimal, error) {
	return decimal.Pow(ctx, base, decimal.NewFromInt64(ctx, int64(n)))
}

// timingFactor returns 1 for PaymentEnd and (1+r) for PaymentBegin —
// the multiplier by which the annuity formula needs to be scaled when
// payments fall at the start of each period.
func timingFactor(ctx decimal.Context, r decimal.Decimal, t PaymentTiming) decimal.Decimal {
	if t == PaymentBegin {
		return onePlus(ctx, r)
	}
	return decimal.NewFromInt64(ctx, 1)
}

// validateNper rejects nper values that aren't strictly positive. The
// TVM equation degenerates at nper <= 0.
func validateNper(nper int) error {
	if nper <= 0 {
		return errors.New("nper must be positive")
	}
	return nil
}

// validateTiming guards against bogus PaymentTiming values that may
// have been built by casting from an int.
func validateTiming(t PaymentTiming) error {
	if t != PaymentEnd && t != PaymentBegin {
		return errors.New("timing must be PaymentEnd or PaymentBegin")
	}
	return nil
}
