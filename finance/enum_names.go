package finance

import "fmt"

// String returns the constant name of v, or DayCountBasis(n) for a value that
// is not one of the constants.
func (d DayCountBasis) String() string {
	switch d {
	case Basis30_360US:
		return "Basis30_360US"
	case BasisActualActual:
		return "BasisActualActual"
	case BasisActual360:
		return "BasisActual360"
	case BasisActual365:
		return "BasisActual365"
	case Basis30_360EU:
		return "Basis30_360EU"
	}
	return fmt.Sprintf("DayCountBasis(%d)", int(d))
}

// String returns the constant name of v, or PaymentTiming(n) for a value that
// is not one of the constants.
func (p PaymentTiming) String() string {
	switch p {
	case PaymentEnd:
		return "PaymentEnd"
	case PaymentBegin:
		return "PaymentBegin"
	}
	return fmt.Sprintf("PaymentTiming(%d)", int(p))
}
