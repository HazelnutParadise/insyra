package excel

import "fmt"

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
