package parquet

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func writeAndRead(t *testing.T, cells ...any) []any {
	t.Helper()
	dl := insyra.NewDataList()
	for _, c := range cells {
		dl.Append(c)
	}
	dt := insyra.NewDataTable(dl)
	path := filepath.Join(t.TempDir(), "b.parquet")
	if err := Write(dt, path); err != nil {
		t.Fatal(err)
	}
	got, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return got.GetColByNumber(0).Data()
}

// A column of byte slices used to be written as the text "[65 45 48 49]".
func TestByteSliceColumnRoundTripsAsBytes(t *testing.T) {
	data := writeAndRead(t, []byte("A-01"), []byte{0xff, 0x00}, nil)
	first, ok := data[0].([]byte)
	if !ok || !bytes.Equal(first, []byte("A-01")) {
		t.Fatalf("first cell = %#v, want the bytes A-01", data[0])
	}
	if second, ok := data[1].([]byte); !ok || !bytes.Equal(second, []byte{0xff, 0x00}) {
		t.Errorf("second cell = %#v, want ff 00", data[1])
	}
	if data[2] != nil {
		t.Errorf("nil cell = %#v, want nil", data[2])
	}
}

// Mixed with text, a byte slice that is text is written as that text.
func TestByteSliceInATextColumnIsItsText(t *testing.T) {
	data := writeAndRead(t, "x", []byte("A-01"))
	if data[1] != "A-01" {
		t.Errorf("cell = %#v, want A-01", data[1])
	}
}
