# Design: fix-api-review-batch-5

## Context

All six were reproduced by the review and pinned by a failing test before any code changed. The shared property: the wrong answer looks like a normal one.

## Decisions

1. **Exact integer comparison in `CompareAny`.** A helper pair (`asInt64`, `asUint64`) plus `compareIntegers` handles signed/signed, unsigned/unsigned and the two mixed cases (a negative signed value is always smaller; otherwise compare as `uint64`). Only when at least one side is a float does the old `float64` path run, so `2` versus `2.5` still works. *Alternative*: compare via `math/big` — rejected, far slower on the sort hot path for no extra correctness in the integer domain.
2. **`Rank` orders on the original cells.** It read `numericCells`' `float64` copies, so the fix in `CompareAny` alone would not have reached it. It now snapshots the cells under the lock and uses `CompareAny` for both ordering and tie detection; the missing-value rule (nil/NaN rank NaN and take no position) is unchanged, still driven by the `float64` view.
3. **Hermite basis.** `H_i = (1 - 2(x-x_i)L'_i(x_i))L_i²` and `Ĥ_i = (x-x_i)L_i²`, with `L'_i(x_i) = Σ_{j≠i} 1/(x_i-x_j)`. The old code used `L_i` (not squared) and dropped the correction factor. Verified three ways in the test: the closed-form cubic for a two-node case, exact reproduction of `x²`, and numerical differentiation at every node.
4. **k-means redraw only on collision.** R draws initial centres from the distinct rows. Switching the pool unconditionally would change the RNG draw for every existing seed and invalidate the cross-language fixtures. Instead the draw is checked afterwards and redrawn from `uniqueRows` only when it actually contained a duplicate — impossible for `NStart >= 2`, which already uses the distinct pool. The redraw is therefore applied only in the single-start path, which also keeps one shared `initPool` for all starts (they index into it).
5. **`TryParseTime` layout list, longest first**, so `2006-01-02 15:04:05` is not truncated by a shorter layout. Layouts without a zone parse as UTC, which is `time.Parse`'s rule; a test pins that plain numbers and words still do not match, because CCL uses this function to probe whether a string is a date.
6. **`ShowRange`: fix the doc, not the code.** A negative end that stays exclusive matches Python slicing, and `ShowRange(2, nil)` already expresses "to the end". Changing the code would leave no way to say "up to but excluding the last".
7. **`ShowTypes` ordering** goes through a shared `sortColIndices` that parses the Excel index; unparseable labels sort last by string so the result is still deterministic.
8. **`Close()` during a queued `AtomicDo`.** The pre-lock path already runs `f` inline when the actor is closed; the post-lock path returned without running it, so a write that merely lost a race disappeared with no error. Both paths now run `f`. `AtomicDoN` keeps its all-or-nothing batch check, which is a different contract.

## Risks / Trade-offs

- [`Rank` is now slower: `CompareAny` per comparison instead of `cmp.Compare` on float64] → a type switch per comparison, negligible against the sort itself; correctness on integers is not optional for a ranking function.
- [A string that `TryParseTime` now recognises becomes a date in CCL] → that is the fix; the test pins that non-dates still fail.
- [k-means results change for a seed whose draw contained a duplicate] → those runs previously returned an error, so there is no result to preserve.
