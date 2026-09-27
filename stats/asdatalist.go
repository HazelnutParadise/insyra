package stats

import "github.com/HazelnutParadise/insyra"

// asDataList turns any IDataList into a concrete *insyra.DataList without an
// unchecked type assertion and without copying it. A concrete list is
// returned as is. Any other implementation is read through the
// *insyra.DataList its AtomicDo hands over, so its cells arrive exactly as
// stored; rebuilding it with NewDataList would flatten a slice cell into
// several numbers. nil, a typed nil, or an implementation that hands over nil
// becomes an empty list, so the caller's existing length checks report the
// error instead of a panic.
func asDataList(dl insyra.IDataList) *insyra.DataList {
	if dl == nil {
		return insyra.NewDataList()
	}
	if concrete, ok := dl.(*insyra.DataList); ok {
		if concrete == nil {
			return insyra.NewDataList()
		}
		return concrete
	}
	var inner *insyra.DataList
	dl.AtomicDo(func(l *insyra.DataList) {
		inner = l
	})
	if inner == nil {
		return insyra.NewDataList()
	}
	return inner
}
