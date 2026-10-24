package nn

import (
	"math/rand"
	"testing"
)

// The masks and initial weights a tape draws must not depend on when its
// generator is created: they are the stream an eagerly seeded source gives.
func TestTapeDrawsTheStreamOfItsSeed(t *testing.T) {
	const probability = float32(0.5)
	input := mustTestTensor(t, []int{64}, make([]float32, 64))
	cases := []struct {
		name string
		tape *Tape
		seed int64
	}{
		{"default seed", NewTape(), 1},
		{"seed 0", NewTape(0), 0},
		{"seed 7", NewTape(7), 7},
		{"negative seed", NewTape(-3), -3},
		{"zero-value tape", &Tape{marked: map[*Tensor]*Parameter{}, grads: map[*Tensor]*Tensor{}}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Two Dropouts in a row read one continuous stream.
			reference := rand.New(rand.NewSource(c.seed))
			got := append(dropoutKeeps(t, c.tape, input, probability), dropoutKeeps(t, c.tape, input, probability)...)
			for i, kept := range got {
				if want := reference.Float32() >= probability; kept != want {
					t.Fatalf("draw %d kept=%v, the seed's stream gives %v", i, kept, want)
				}
			}
		})
	}
}

// dropoutKeeps runs one Dropout on a tensor of ones and reports which values
// it kept.
func dropoutKeeps(t *testing.T, tape *Tape, like *Tensor, probability float32) []bool {
	t.Helper()
	ones := make([]float32, len(like.data))
	for i := range ones {
		ones[i] = 1
	}
	out, err := tape.Dropout(mustTestTensor(t, like.Shape(), ones), probability)
	if err != nil {
		t.Fatal(err)
	}
	keeps := make([]bool, len(out.data))
	for i, v := range out.data {
		keeps[i] = v != 0
	}
	return keeps
}

// A tape that never draws must not seed a generator.
func TestNewTapeSeedsNothingUntilItDraws(t *testing.T) {
	tape := NewTape(42)
	if tape.rng != nil {
		t.Fatal("NewTape created its generator before anything drew from it")
	}
	if _, err := tape.Dropout(mustTestTensor(t, []int{1}, []float32{1}), 0.5); err != nil {
		t.Fatal(err)
	}
	if tape.rng == nil {
		t.Fatal("Dropout did not create the generator")
	}
}

var benchmarkTape *Tape

func BenchmarkNewTape(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchmarkTape = NewTape()
	}
}
