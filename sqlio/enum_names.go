package sqlio

import "fmt"

// String returns the constant name of p, or TableExistsPolicy(n) for a value
// that is not one of the constants.
func (p TableExistsPolicy) String() string {
	switch p {
	case TableExistsFail:
		return "TableExistsFail"
	case TableExistsReplace:
		return "TableExistsReplace"
	case TableExistsAppend:
		return "TableExistsAppend"
	}
	return fmt.Sprintf("TableExistsPolicy(%d)", int(p))
}
