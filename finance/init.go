// Package finance provides high-precision financial calculations
// (annuities, loans, NPV/IRR, rate conversions, amortization schedules)
// built on top of github.com/TimLai666/go-decimal.
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
