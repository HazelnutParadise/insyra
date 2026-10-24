# Design: lazy-tape-rng

## Decisions

**The seed lives on the tape, the generator is made on first use.** `Tape` gains `seed int64` and `seedSet bool`. `NewTape` sets both and leaves `rng` nil. A private `random()` returns `rng`, creating `rand.New(rand.NewSource(seed))` the first time, or with the default seed 1 when `seedSet` is false (a zero `Tape`). `Dropout` and the four layer initialisers that used to do `if t.rng == nil { t.rng = newDefaultTapeRNG() }` call `random()` instead, so there is one place that decides the seed.

`seedSet` is needed because 0 is a valid seed: `NewTape(0)` must draw from seed 0, while a zero `Tape` draws from seed 1, as it did before.

**`Sequential.Fit` is unchanged.** It assigns `tape.rng` directly from its own seed, which `random()` then returns as is.

**No `Clear` change.** `Clear` never touched the generator; a cleared tape keeps drawing where it left off, before and after.

## Verification

- `TestTapeDrawsTheStreamOfItsSeed` compares two consecutive `Dropout` masks against a reference `rand.New(rand.NewSource(seed))` for the default seed, seeds 0, 7 and -3, and a zero `Tape`. It passes on the old code too, which is what makes it the bit-identity proof.
- `TestNewTapeSeedsNothingUntilItDraws` fails on the old code and passes on the new.
- The whole `nn` suite, which pins Fit's MNIST curve and the dropout determinism tests, passes unchanged.
- `BenchmarkNewTape`, alternated with an untouched `origin/dev` checkout on the M3.
