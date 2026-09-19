package stats

import "github.com/HazelnutParadise/insyra"

// asDataList turns any IDataList into a concrete *insyra.DataList without an
// unchecked type assertion. A concrete list is returned as is; any other
// implementation is copied through Data(); nil becomes an empty list so the
// caller's existing length checks report the error instead of a panic.
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
	return insyra.NewDataList(dl.Data()...)
}
