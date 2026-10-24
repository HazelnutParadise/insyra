# Proposal: lazy-tape-rng

## Why

`nn.NewTape` builds `rand.New(rand.NewSource(seed))` on every call, but only a draw (`Dropout`, or a layer's random initialisation) ever reads the source. Seeding the 607-word lagged-Fibonacci state is most of what a new tape costs: measured on an M3 on 2026-10-24, `BenchmarkNewTape` took 13–34 µs and 5,712 B per tape. CoImNet builds one short-lived tape per sequence of every minibatch to score its loss, so about 4% of a training epoch's CPU samples went into `rngSource.Seed` ([#536](https://github.com/HazelnutParadise/insyra/issues/536)).

## What Changes

- `NewTape` records its seed and leaves the generator unset. The tape creates the generator from that seed the first time something draws from it, through one private accessor that `Dropout` and the layer initialisers share.
- A zero `Tape{}` keeps drawing from the default seed 1, as it did through `newDefaultTapeRNG`, which goes away.
- Every seed, including the default and seed 0, produces the same stream as before, so masks and initial weights are unchanged. `Tape.Clear` does not touch the generator, as before. `Sequential.Fit` still installs its own generator.

## Capabilities

### Modified Capabilities

- `nn-training`: a tape states that it seeds its generator on first draw and that the stream is the one its seed gives.

## Impact

`nn/autodiff.go`, `nn/autodiff_practice.go`, `nn/layers.go`, `nn/layers_attention.go`, `nn/layers_catalog.go`, the new `nn/tape_rng_test.go`, both CHANGELOGs. No exported API changes.
