package wgpu

import (
	"context"
	"fmt"
	"testing"
)

// A clone made right after a buffer is written reads what was written, and so
// does a clone made right after a readback, on every backend (#626: on
// Vulkan, before the write was settled, the clone read zeros).
func TestCloneRightAfterAWriteReadsTheWrite(t *testing.T) {
	requireResidentDevice(t)
	for round := range 20 {
		values := make([]float32, 320)
		for i := range values {
			values[i] = float32(round*1000+i) - 0.25
		}
		up, err := Upload(values)
		if err != nil {
			t.Fatal(err)
		}
		if round%2 == 1 {
			// A readback just before the next upload, as in the report.
			if _, err := up.Download(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := up.Write(values); err != nil {
				t.Fatal(err)
			}
		}
		clone, err := Clone(up)
		if err != nil {
			t.Fatal(err)
		}
		got, err := clone.Download(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for i, v := range got {
			if v != values[i] {
				up.Release()
				clone.Release()
				t.Fatalf("round %d: clone[%d] = %v, want %v", round, i, v, values[i])
			}
		}
		// A clone of the clone, read straight away.
		second, err := Clone(clone)
		if err != nil {
			t.Fatal(err)
		}
		got, err = second.Download(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprint(got[:4]) != fmt.Sprint(values[:4]) || got[len(got)-1] != values[len(values)-1] {
			t.Fatalf("round %d: clone of a clone = %v..., want %v...", round, got[:4], values[:4])
		}
		up.Release()
		clone.Release()
		second.Release()
	}
}
