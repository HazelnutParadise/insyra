package insyra_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

func TestIsNilList(t *testing.T) {
	var typedNil *insyra.DataList
	cases := []struct {
		name string
		dl   insyra.IDataList
		want bool
	}{
		{"nil interface", nil, true},
		{"typed nil", typedNil, true},
		{"isr.DL zero value", isr.DL, true},
		{"isr wrapper of nil", isr.UseDL((*insyra.DataList)(nil)), true},
		{"real list", insyra.NewDataList(1, 2), false},
		{"isr wrapper of a real list", isr.DL.From(1, 2), false},
	}
	for _, c := range cases {
		if got := insyra.IsNilList(c.dl); got != c.want {
			t.Errorf("%s: IsNilList = %v, want %v", c.name, got, c.want)
		}
	}
}
