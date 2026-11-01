package nn

import "testing"

// A device copy of a tensor made right after it was uploaded holds the
// uploaded values (#626: on Vulkan the copy read zeros).
func TestDeviceCopyRightAfterUpload(t *testing.T) {
	dev := residentTestDevice(t)
	for round := range 10 {
		values := make([]float32, 20*16)
		for i := range values {
			values[i] = float32(i+round) - 0.25
		}
		seed, err := NewTensor([]int{20, 16}, values)
		if err != nil {
			t.Fatal(err)
		}
		if round%2 == 1 {
			// A readback just before the upload, as in the report's training path.
			if _, err := seed.ToHost(); err != nil {
				t.Fatal(err)
			}
		}
		up, err := seed.To(dev)
		if err != nil {
			t.Fatal(err)
		}
		clone, err := up.To(dev)
		if err != nil {
			t.Fatal(err)
		}
		host, err := clone.ToHost()
		if err != nil {
			t.Fatal(err)
		}
		got, err := host.Float32Data()
		if err != nil {
			t.Fatal(err)
		}
		for i, v := range got {
			if v != values[i] {
				t.Fatalf("round %d: copy[%d] = %v, want %v", round, i, v, values[i])
			}
		}
	}
}
