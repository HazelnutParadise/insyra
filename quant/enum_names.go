package quant

import "fmt"

// String returns the constant name of v, or OptionType(n) for a value that
// is not one of the constants.
func (o OptionType) String() string {
	switch o {
	case OptionCall:
		return "OptionCall"
	case OptionPut:
		return "OptionPut"
	}
	return fmt.Sprintf("OptionType(%d)", int(o))
}

// String returns the constant name of v, or PortfolioObjective(n) for a value that
// is not one of the constants.
func (p PortfolioObjective) String() string {
	switch p {
	case MinimumVariance:
		return "MinimumVariance"
	case TargetReturn:
		return "TargetReturn"
	case MaximumSharpe:
		return "MaximumSharpe"
	}
	return fmt.Sprintf("PortfolioObjective(%d)", int(p))
}

// String returns the constant name of v, or VaRMethod(n) for a value that
// is not one of the constants.
func (v VaRMethod) String() string {
	switch v {
	case VaRHistorical:
		return "VaRHistorical"
	case VaRParametric:
		return "VaRParametric"
	}
	return fmt.Sprintf("VaRMethod(%d)", int(v))
}

// String returns the constant name of s, or ResamplingScheme(n) for a value
// that is not one of the constants.
func (s ResamplingScheme) String() string {
	switch s {
	case ResampleStationary:
		return "ResampleStationary"
	case ResampleMovingBlock:
		return "ResampleMovingBlock"
	}
	return fmt.Sprintf("ResamplingScheme(%d)", int(s))
}

// String returns the constant name of m, or AbnormalModel(n) for a value that
// is not one of the constants.
func (m AbnormalModel) String() string {
	switch m {
	case AbnormalMarketAdjusted:
		return "AbnormalMarketAdjusted"
	case AbnormalMarketModel:
		return "AbnormalMarketModel"
	}
	return fmt.Sprintf("AbnormalModel(%d)", int(m))
}
