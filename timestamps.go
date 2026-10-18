package insyra

import "time"

// clock is where DataList and DataTable read the time they record. Tests swap
// it to set the time instead of sleeping across a second boundary.
var clock = time.Now

// timeFromUnixNano turns a stored timestamp back into a time. A value that was
// never set reads as the zero time, so IsZero reports it.
func timeFromUnixNano(ns int64) time.Time {
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}
