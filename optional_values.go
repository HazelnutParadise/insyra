package insyra

import "fmt"

// extraOptional returns the failure for a trailing optional parameter given
// more than one value, or "" when it got at most one. A trailing ...T or
// ...XxxOptions stands for one optional value; keeping the first and dropping
// the rest would hide from the caller that part of what they passed was
// ignored.
func extraOptional(what string, n int) string {
	if n <= 1 {
		return ""
	}
	return fmt.Sprintf("at most one %s may be given, got %d", what, n)
}
