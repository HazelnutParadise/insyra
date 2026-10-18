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

// String returns the constant name of v, or LabelSort(n) for a value that
// is not one of the constants.
func (l LabelSort) String() string {
	switch l {
	case LabelSortFirstSeen:
		return "LabelSortFirstSeen"
	case LabelSortLexicographic:
		return "LabelSortLexicographic"
	case LabelSortByFrequency:
		return "LabelSortByFrequency"
	}
	return fmt.Sprintf("LabelSort(%d)", int(l))
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

// String returns the constant name of v, or NaNPolicy(n) for a value that
// is not one of the constants.
func (n NaNPolicy) String() string {
	switch n {
	case NaNAsCategory:
		return "NaNAsCategory"
	case NaNError:
		return "NaNError"
	case NaNSkip:
		return "NaNSkip"
	}
	return fmt.Sprintf("NaNPolicy(%d)", int(n))
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

// String returns the constant name of v, or SQLActionIfTableExists(n) for a value that
// is not one of the constants.
func (s SQLActionIfTableExists) String() string {
	switch s {
	case SQLActionIfTableExistsFail:
		return "SQLActionIfTableExistsFail"
	case SQLActionIfTableExistsReplace:
		return "SQLActionIfTableExistsReplace"
	case SQLActionIfTableExistsAppend:
		return "SQLActionIfTableExistsAppend"
	}
	return fmt.Sprintf("SQLActionIfTableExists(%d)", int(s))
}

// String returns the constant name of v, or SheetExistsPolicy(n) for a value that
// is not one of the constants.
func (s SheetExistsPolicy) String() string {
	switch s {
	case SheetExistsFail:
		return "SheetExistsFail"
	case SheetExistsReplace:
		return "SheetExistsReplace"
	}
	return fmt.Sprintf("SheetExistsPolicy(%d)", int(s))
}

// String returns the constant name of v, or UnknownPolicy(n) for a value that
// is not one of the constants.
func (u UnknownPolicy) String() string {
	switch u {
	case UnknownIgnore:
		return "UnknownIgnore"
	case UnknownError:
		return "UnknownError"
	case UnknownAsNew:
		return "UnknownAsNew"
	}
	return fmt.Sprintf("UnknownPolicy(%d)", int(u))
}
