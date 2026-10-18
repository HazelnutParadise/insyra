# Tasks

- [x] Reproduce #441 on `dev`: `[]float32` and `[]int` buffers return an empty result and a nil error.
- [x] `ExecuteNearestExact` checks every buffer before planning and returns an error naming the column (`accel/exact.go` `checkExactBuffers`).
- [x] Test: unreadable values, nil values, type disagreement, ragged columns (`accel/exact_buffers_test.go`); `TestIneligibleDatasetReturnsNothing` still passes.
- [x] `go test ./accel/...` passes.
- [x] `Docs/accel.md` states the refusal.
- [x] Entries under `## Unreleased` in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] `openspec validate accel-refuse-unreadable-buffers --strict` passes.
