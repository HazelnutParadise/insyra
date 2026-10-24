package nn

import "testing"

func TestNewTapeBuildsItsRandomSourceOnFirstUse(t *testing.T) {
	tape := NewTape(7)
	if tape.rng != nil {
		t.Fatal("NewTape seeded a random source before anything drew from it")
	}
	input, err := NewTensor([]int{64}, make([]float32, 64))
	if err != nil {
		t.Fatal(err)
	}
	for i := range input.data {
		input.data[i] = 1
	}
	first, err := tape.Dropout(input, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if tape.rng == nil {
		t.Fatal("Dropout did not build the random source")
	}
	// The same seed gives the same mask whether the source was built eagerly or on use.
	again, err := NewTape(7).Dropout(input, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	for i := range first.data {
		if first.data[i] != again.data[i] {
			t.Fatalf("mask differs at %d for the same seed", i)
		}
	}
	other, err := NewTape(8).Dropout(input, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	same := true
	for i := range first.data {
		if first.data[i] != other.data[i] {
			same = false
		}
	}
	if same {
		t.Fatal("different seeds gave the same mask")
	}
}

func BenchmarkNewTape(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = NewTape()
	}
}
