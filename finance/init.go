// Package finance provides high-precision financial calculations
// (annuities, loans, NPV/IRR, rate conversions, amortization schedules)
// built on top of github.com/TimLai666/go-decimal.
//
// The API mirrors Excel/Google-Sheets function names and signatures for
// familiarity, but each result follows the financially-correct formula rather
// than being defined as "whatever Excel returns". Where a spreadsheet diverges
// from the correct convention, this package follows the correct one — e.g.
// TBillEq uses the US Treasury / SIA coupon-equivalent yield for bills longer
// than 182 days, whereas Excel's TBILLEQ keeps its short-bill formula and
// overstates the yield there. Outputs are validated against numpy_financial and
// QuantLib; TBillEq beyond 182 days is currently the only such divergence.
//
// All exported functions use the Excel/Google-Sheets sign convention:
// money received is positive, money paid out is negative. PMT, FV, and PV
// therefore typically come back with the opposite sign of one another.
//
// Precision and rounding are configurable per call via the optional
// Options argument; when omitted, results are produced at scale=10 with
// HalfUp rounding (internal computation always uses extra guard digits).
package finance

func init() {}
