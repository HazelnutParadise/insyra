# Proposal: fix-api-review-batch-5

## Why

Six findings from the whole-repo review produce silently wrong answers rather than errors — the worst category for a data library, because nothing tells the user. Routing every integer through `float64` makes any two `int64` above 2^53 compare equal, so `Sort`, `Rank`, `SortBy`, `Pivot` and `Describe`'s min/max all order them wrongly. `HermiteInterpolation` uses the wrong basis and satisfies none of its derivative conditions. `KMeans` fails on 44 of 50 seeds when the data has repeated rows. `TryParseTime` does not recognise `2006-01-02 15:04:05`, so CCL's date functions and `datafetch` quietly see a string. `ShowTypes` prints columns in the wrong order past column 26 and `ShowRange`'s documented behaviour does not match the code. `Close()` silently drops an operation that was already waiting for the lock. None needs an API decision. Closes #332, #333, #334, #335, #336, #331.

## What Changes

- `CompareAny` compares two integer values exactly (including mixed signed/unsigned and values outside `int64`), falling back to `float64` only when one side is a float. `DataList.Rank` orders and detects ties on the original cells instead of `float64` copies, for the same reason.
- `HermiteInterpolation` uses the standard Hermite basis, so the result passes through every value and matches every supplied derivative.
- `KMeans` redraws its initial centres from the distinct rows when a draw picked the same row twice. Only that case changes, so every existing seeded result stays bit-identical.
- `TryParseTime` accepts the common zone-less layouts (`2006-01-02 15:04:05`, `2006-01-02T15:04:05`, `2006-01-02 15:04`) and `/` separators, still rejecting non-dates.
- `ShowTypes` orders columns by position, matching `Show`. `ShowRange`'s doc now states the Python-slice rule the code implements (a negative end is exclusive; pass `nil` to reach the end).
- `AtomicDo` runs a callback that was already queued on the mutex even if `Close()` happened meanwhile: Close drops the locking, never the operation.

## Capabilities

### New Capabilities

- `exact-numeric-ordering`: ordering and ranking never lose precision on integers.
- `interpolation-correctness`: an interpolant satisfies the conditions it is named for.
- `clustering-initial-centers`: k-means picks distinct initial centres, as R does.
- `date-parsing-coverage`: the common timestamp layouts are recognised.
- `display-ordering`: every display path shows columns in their real order.
- `actor-close-semantics`: Close stops locking, not queued work.

### Modified Capabilities

(none)

## Impact

- `internal/algorithms/sort.go`, `internal/algorithms/interpolation.go`, `internal/utils/utils.go`, `internal/core/atomic.go`, `datalist.go` (`Rank`), `show.go`, `stats/internal/clustering/cluster.go`, plus tests and docs.
- No signature changes. `Rank` and `Sort` results change only where `float64` was previously losing the distinction; `TryParseTime` newly recognising a layout makes CCL treat such a string as a date, which is the point of the fix.
