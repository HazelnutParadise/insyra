package stats

import (
	"testing"

	"github.com/HazelnutParadise/insyra/isr"
)

func TestAsDataListTurnsAWrapperOfANilListIntoAnEmptyOne(t *testing.T) {
	if got := asDataList(isr.DL); got == nil || got.Len() != 0 {
		t.Fatalf("want an empty list, got %v", got)
	}
}
