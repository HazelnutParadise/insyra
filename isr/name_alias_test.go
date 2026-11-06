package isr_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

// isr.Name and insyra.Name build the same selector, as on 0.4, so either
// works wherever the library takes a name.
func TestIsrNameIsInsyraName(t *testing.T) {
	selector := isr.Name("price")
	var _ insyra.NameSelector = selector // the same type, not a conversion
	if selector != insyra.Name("price") || selector.Value() != "price" {
		t.Fatalf("isr.Name(%q) = %#v", "price", selector)
	}
	dt := isr.DT.From(isr.DL.From(1, 2).SetName("price"))
	if got := dt.Col(insyra.Name("price")).Data(); len(got) != 2 {
		t.Fatalf("DT.Col(insyra.Name) = %v", got)
	}
}
