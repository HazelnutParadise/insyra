package insyra

import (
	"testing"
)

func TestProcessData(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	s, len := ProcessData(dl)
	if len != 3 {
		t.Errorf("ProcessData() did not return the correct length")
	}
	if s == nil {
		t.Errorf("ProcessData() did not return the correct slice")
	}
}
