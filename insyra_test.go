package insyra_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestProcessData(t *testing.T) {
	dl := insyra.NewDataList(1, 2, 3)
	s, len := insyra.ProcessData(dl)
	if len != 3 {
		t.Errorf("ProcessData() did not return the correct length")
	}
	if s == nil {
		t.Errorf("ProcessData() did not return the correct slice")
	}
}
