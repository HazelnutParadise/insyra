package stats

import "fmt"

// String returns the constant name of v, or CorrelationMethod(n) for a value that
// is not one of the constants.
func (c CorrelationMethod) String() string {
	switch c {
	case PearsonCorrelation:
		return "PearsonCorrelation"
	case KendallCorrelation:
		return "KendallCorrelation"
	case SpearmanCorrelation:
		return "SpearmanCorrelation"
	}
	return fmt.Sprintf("CorrelationMethod(%d)", int(c))
}

// String returns the constant name of v, or KurtosisMethod(n) for a value that
// is not one of the constants.
func (k KurtosisMethod) String() string {
	switch k {
	case KurtosisG2:
		return "KurtosisG2"
	case KurtosisAdjusted:
		return "KurtosisAdjusted"
	case KurtosisBiasAdjusted:
		return "KurtosisBiasAdjusted"
	}
	return fmt.Sprintf("KurtosisMethod(%d)", int(k))
}

// String returns the constant name of v, or SkewnessMethod(n) for a value that
// is not one of the constants.
func (s SkewnessMethod) String() string {
	switch s {
	case SkewnessG1:
		return "SkewnessG1"
	case SkewnessAdjusted:
		return "SkewnessAdjusted"
	case SkewnessBiasAdjusted:
		return "SkewnessBiasAdjusted"
	}
	return fmt.Sprintf("SkewnessMethod(%d)", int(s))
}

// String returns the constant name of c, or CovType(n) for a value that is not
// one of the constants.
func (c CovType) String() string {
	switch c {
	case CovOLS:
		return "CovOLS"
	case CovHC0:
		return "CovHC0"
	case CovHC1:
		return "CovHC1"
	case CovHC2:
		return "CovHC2"
	case CovHC3:
		return "CovHC3"
	case CovHAC:
		return "CovHAC"
	}
	return fmt.Sprintf("CovType(%d)", int(c))
}
