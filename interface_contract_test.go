package insyra

import (
	"reflect"
	"sort"
	"testing"
)

// IDataList and IDataTable list every exported method of *DataList and
// *DataTable, so code written against the interface can do anything the
// concrete type can. The exceptions are the methods an extension in this
// module overrides with a signature of its own; listed, they would make that
// extension fail the interface and keep it out of stats, plot and Merge.
//
//   - ClearErr and SetErr return the receiver so a chain can continue; isr
//     overrides them to return its own type, so
//     `isr.DT.From(x).ClearErr().Push(row)` keeps isr's methods. Err and
//     PopErr return no receiver and stay listed.
//   - Pivot and Unpivot are overridden by isr's short-syntax versions, which
//     take isr.Pivot / isr.Unpivot and return the isr table for chaining.
//
// Owner's ruling of 2026-09-26 (#208): the interfaces list everything, and
// any exception is written here with its reason.
var interfaceExceptions = map[string]string{
	"ClearErr": "isr overrides it to return its own type for chaining",
	"SetErr":   "isr overrides it to return its own type for chaining",
	"Pivot":    "isr overrides it with a short-syntax version",
	"Unpivot":  "isr overrides it with a short-syntax version",
}

func TestInterfacesListEveryMethod(t *testing.T) {
	check := func(concrete, iface reflect.Type) {
		listed := map[string]bool{}
		for i := 0; i < iface.NumMethod(); i++ {
			listed[iface.Method(i).Name] = true
		}
		var missing, wronglyListed []string
		for i := 0; i < concrete.NumMethod(); i++ {
			name := concrete.Method(i).Name
			switch {
			case interfaceExceptions[name] != "" && listed[name]:
				wronglyListed = append(wronglyListed, name)
			case interfaceExceptions[name] == "" && !listed[name]:
				missing = append(missing, name)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("%s is missing %v; add them, or list them in interfaceExceptions with the reason", iface, missing)
		}
		if len(wronglyListed) > 0 {
			t.Errorf("%s lists %v, which it leaves out on purpose so an embedding type can override them", iface, wronglyListed)
		}
	}
	check(reflect.TypeOf(&DataTable{}), reflect.TypeOf((*IDataTable)(nil)).Elem())
	check(reflect.TypeOf(&DataList{}), reflect.TypeOf((*IDataList)(nil)).Elem())
}
