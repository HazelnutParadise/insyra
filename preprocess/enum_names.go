package preprocess

import "fmt"

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
