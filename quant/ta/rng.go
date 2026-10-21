package ta

import "math/rand/v2"

// seedXor mirrors the core sampling seeding so one uint64 seeds both PCG words.
const seedXor uint64 = 0x9E3779B97F4A7C15

// seededRNG wraps a PCG source and performs the uniform reduction itself, so the
// output depends only on the PCG sequence, a published algorithm, and not on
// the standard library's IntN, whose stability across Go releases is not
// documented. It is the same generator quant's bootstrap uses.
type seededRNG struct {
	src *rand.PCG
}

func newSeededRNG(seed uint64) *seededRNG {
	return &seededRNG{src: rand.NewPCG(seed, seed^seedXor)}
}

// uintN returns a uniform integer in [0, n) by rejection sampling. n > 0.
func (r *seededRNG) uintN(n uint64) uint64 {
	threshold := -n % n // (2^64 mod n): values below it would bias the modulo
	for {
		v := r.src.Uint64()
		if v >= threshold {
			return v % n
		}
	}
}
