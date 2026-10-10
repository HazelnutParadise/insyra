package insyra

import (
	"fmt"
	"io"
	"os"
)

// showCountIsPositive reports whether n is a count ShowHead or ShowTail can
// show. When it is not, it writes an error line to w instead: passing n on to
// ShowRange would turn ShowHead(-3) into the last three rows and ShowTail(-3)
// into the first three.
func showCountIsPositive(w io.Writer, method, unit string, n int) bool {
	if n > 0 {
		return true
	}
	fmt.Fprintln(w, colorText("1;31", fmt.Sprintf("ERROR: %s needs a positive number of %s, got %d", method, unit, n)))
	return false
}

// ShowHead displays the first n rows, the same as ShowRange(n).
func (dt *DataTable) ShowHead(n int) {
	dt.ShowHeadTo(os.Stdout, n)
}

// ShowHeadTo is ShowHead writing to w instead of os.Stdout.
func (dt *DataTable) ShowHeadTo(w io.Writer, n int) {
	if !showCountIsPositive(w, "ShowHead", "rows", n) {
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
	if !showCountIsPositive(w, "ShowTail", "rows", n) {
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
	if !showCountIsPositive(w, "ShowHead", "items", n) {
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
	if !showCountIsPositive(w, "ShowTail", "items", n) {
		return
	}
	dl.ShowRangeTo(w, -n)
}
