# Tasks: core-utils-cleanup

## 1. Tests first

- [x] 1.1 `utils_test.go`: `ProcessData` reads a DataList, a slice, an array and pointers to both, and refuses `nil`, `42`, a string, a map, a nil `*DataList` and a nil slice pointer with an error and a nil slice. Against the old code the file does not compile, because `ProcessData` returned an int.
- [x] 1.2 `utils_test.go`: `SqrtRat` returns nil for -1 and nil, and 3/2 for 9/4; `PowRat` gives 9/4 for (2/3, -2), -1/8 for (-2, -3), nil for (0, -1) and (nil, 2), and its result does not alias its base. Against the old code `SqrtRat(-1)` panicked with `square root of negative operand`, `PowRat(2/3, -2)` and `PowRat(-2, -3)` returned 1/1, `PowRat(0, -1)` returned 1/1, and `PowRat(nil, 2)` panicked.
- [x] 1.3 `stats/moment_input_test.go`: `Skewness` and `Kurtosis` refuse a nil `*DataList` and `42` with ProcessData's error, not `empty data`. Against the old code the nil list crashed with a nil-pointer dereference.
- [x] 1.4 `utils_test.go`: `WeightedMean(42)` and `WeightedMovingAverage(2, "ab")` record `weights: cannot read …`; a DataList of weights still works.

## 2. Implementation

- [x] 2.1 `utils.go`: `ProcessData` returns `([]any, error)`; `SqrtRat` and `PowRat` never panic and compute negative powers; `SqrtRat`, `PowRat`, `SortTimes` and `F64orRat` are Deprecated with their replacement named.
- [x] 2.2 `datalist.go`: `WeightedMovingAverage` and `WeightedMean` report the weights error; `WeightedMean` reads its weights before `AtomicDo`.
- [x] 2.3 `stats/skewness.go`, `stats/kurtosis.go`: wrap ProcessData's error as `sample: …`; `mkt/cai.go`: `slices.SortFunc(times, time.Time.Compare)`.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/utils.md`: `ProcessData`'s new shape; `SqrtRat`, `PowRat` and `SortTimes` marked Deprecated with their replacements and their nil results; the `SliceToF64` line that said only `float64` and `int` convert; the usage example no longer calls `SqrtRat`. `Docs/DataList.md`: what `WeightedMean` and `WeightedMovingAverage` accept and do when the weights cannot be read. `Docs/stats.md`: the input `Skewness` and `Kurtosis` accept.
- [x] 3.2 Skills: no change. Neither skill names these helpers, and the two error shapes they teach are unchanged.
- [x] 3.3 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core (BREAKING `ProcessData`; the helpers deprecated and fixed) and `stats` (Skewness and Kurtosis).
- [x] 3.4 `api-review.md`: K-15 and IN-17 marked fixed.
- [x] 3.5 `AGENTS.md`: a follow-up to remove the four deprecated names; `Skewness` and `Kurtosis` leave the nil-list follow-up. `delivery-status.md`: a Latest Milestones entry.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` (0 issues), `openspec validate core-utils-cleanup --strict`.
