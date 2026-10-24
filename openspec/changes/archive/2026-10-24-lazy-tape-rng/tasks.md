# Tasks

## 1. Implementation

- [x] 1.1 `NewTape` records the seed, and a private `Tape.random()` creates the generator on first use (`nn/autodiff.go`).
- [x] 1.2 `Dropout` and the layer initialisers draw through `random()`; `newDefaultTapeRNG` is removed (`nn/autodiff_practice.go`, `nn/layers.go`, `nn/layers_attention.go`, `nn/layers_catalog.go`).

## 2. Verification

- [x] 2.1 Test that the masks follow the seed's stream for the default seed, seeds 0, 7, -3 and a zero `Tape`, passing before and after the change (`nn/tape_rng_test.go`).
- [x] 2.2 Test that `NewTape` holds no generator until the first draw, failing before the change.
- [x] 2.3 `BenchmarkNewTape` alternated against untouched `origin/dev` on the M3, numbers in the proposal and the changelog.
- [x] 2.4 `go test ./...`, `go vet ./...` and `golangci-lint run` pass.

## 3. Documentation

- [x] 3.1 Entries under `## Unreleased` in both CHANGELOGs.
- [x] 3.2 `openspec validate lazy-tape-rng --strict` passes.
