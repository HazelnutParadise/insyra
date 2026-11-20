# AGENTS.md (gplot/)

Rules for working in `gplot/`. The top-level [AGENTS.md](/AGENTS.md) covers the wider repo.

## Known pitfalls

- Run the tests of code that feeds floats into gonum on amd64 as well (`GOARCH=amd64 go test` on an arm64 host), the architecture of CI's ubuntu leg: gonum's `NewHist` on a NaN and a heat map holding +Inf panicked only on amd64.
