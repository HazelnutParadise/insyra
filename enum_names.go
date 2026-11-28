package insyra

import "fmt"

// String returns the constant name of v, or ErrPoppingMode(n) for a value that
// is not one of the constants.
func (e ErrPoppingMode) String() string {
	switch e {
	case ErrPoppingModeFIFO:
		return "ErrPoppingModeFIFO"
	case ErrPoppingModeLIFO:
		return "ErrPoppingModeLIFO"
	}
	return fmt.Sprintf("ErrPoppingMode(%d)", int(e))
}

// String returns the constant name of v, or MergeDirection(n) for a value that
// is not one of the constants.
func (m MergeDirection) String() string {
	switch m {
	case MergeDirectionHorizontal:
		return "MergeDirectionHorizontal"
	case MergeDirectionVertical:
		return "MergeDirectionVertical"
	}
	return fmt.Sprintf("MergeDirection(%d)", int(m))
}

// String returns the constant name of v, or MergeMode(n) for a value that
// is not one of the constants.
func (m MergeMode) String() string {
	switch m {
	case MergeModeInner:
		return "MergeModeInner"
	case MergeModeOuter:
		return "MergeModeOuter"
	case MergeModeLeft:
		return "MergeModeLeft"
	case MergeModeRight:
		return "MergeModeRight"
	}
	return fmt.Sprintf("MergeMode(%d)", int(m))
}

// String returns the constant name of v, or ResampleFreq(n) for a value that
// is not one of the constants.
func (r ResampleFreq) String() string {
	switch r {
	case ResampleWeekly:
		return "ResampleWeekly"
	case ResampleMonthly:
		return "ResampleMonthly"
	case ResampleQuarterly:
		return "ResampleQuarterly"
	case ResampleYearly:
		return "ResampleYearly"
	}
	return fmt.Sprintf("ResampleFreq(%d)", int(r))
}
