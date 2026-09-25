package insyra

import (
	"fmt"
	"io"
	"os"
)

// showRangeProblem says what is wrong with the arguments of a ShowRange-style
// call, or "" when they can be read: nothing, a count, or a start and an end
// (an int, or nil for the end). Anything else used to be ignored and the whole
// table shown, so the caller never learned the range was not theirs.
func showRangeProblem(startEnd []any) string {
	if len(startEnd) > 2 {
		return fmt.Sprintf("takes at most two values, a start and an end, got %d", len(startEnd))
	}
	if len(startEnd) >= 1 {
		if _, ok := startEnd[0].(int); !ok {
			return fmt.Sprintf("the first value must be an int (a count, or the start), got %T", startEnd[0])
		}
	}
	if len(startEnd) == 2 && startEnd[1] != nil {
		if _, ok := startEnd[1].(int); !ok {
			return fmt.Sprintf("the end must be an int, or nil for the end, got %T", startEnd[1])
		}
	}
	return ""
}

func printShowProblem(w io.Writer, method, problem string) {
	fmt.Fprintln(w, colorText("1;31", fmt.Sprintf("ERROR: %s %s", method, problem)))
}

// ShowHead displays the first n rows, the same as ShowRange(n).
func (dt *DataTable) ShowHead(n int) {
	dt.ShowHeadTo(os.Stdout, n)
}

// ShowHeadTo is ShowHead writing to w instead of os.Stdout.
func (dt *DataTable) ShowHeadTo(w io.Writer, n int) {
	if n <= 0 {
		printShowProblem(w, "ShowHead", fmt.Sprintf("needs a positive number of rows, got %d", n))
		return
	}
	dt.ShowRangeTo(w, n)
}

// ShowTail displays the last n rows, the same as ShowRange(-n).
func (dt *DataTable) ShowTail(n int) {
	dt.ShowTailTo(os.Stdout, n)
}

// ShowTailTo is ShowTail writing to w instead of os.Stdout.
func (dt *DataTable) ShowTailTo(w io.Writer, n int) {
	if n <= 0 {
		printShowProblem(w, "ShowTail", fmt.Sprintf("needs a positive number of rows, got %d", n))
		return
	}
	dt.ShowRangeTo(w, -n)
}

// ShowHead displays the first n items, the same as ShowRange(n).
func (dl *DataList) ShowHead(n int) {
	dl.ShowHeadTo(os.Stdout, n)
}

// ShowHeadTo is ShowHead writing to w instead of os.Stdout.
func (dl *DataList) ShowHeadTo(w io.Writer, n int) {
	if n <= 0 {
		printShowProblem(w, "ShowHead", fmt.Sprintf("needs a positive number of items, got %d", n))
		return
	}
	dl.ShowRangeTo(w, n)
}

// ShowTail displays the last n items, the same as ShowRange(-n).
func (dl *DataList) ShowTail(n int) {
	dl.ShowTailTo(os.Stdout, n)
}

// ShowTailTo is ShowTail writing to w instead of os.Stdout.
func (dl *DataList) ShowTailTo(w io.Writer, n int) {
	if n <= 0 {
		printShowProblem(w, "ShowTail", fmt.Sprintf("needs a positive number of items, got %d", n))
		return
	}
	dl.ShowRangeTo(w, -n)
}
