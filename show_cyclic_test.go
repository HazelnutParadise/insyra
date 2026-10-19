package insyra

import (
	"bytes"
	"strings"
	"testing"
)

func TestShowDoesNotOverflowOnASelfReferentialCell(t *testing.T) {
	cyclic := make([]any, 1)
	cyclic[0] = cyclic
	dl := NewDataList(1)
	dl.Append(cyclic)
	var out bytes.Buffer
	dl.ShowTo(&out)
	if !strings.Contains(out.String(), "[]interface {}") {
		t.Fatalf("want the cell shown by its type, got:\n%s", out.String())
	}
}
