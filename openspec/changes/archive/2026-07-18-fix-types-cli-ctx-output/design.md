# Design — route `types` through ctx.Output

## Approach: replicate M20, no new ideas

M20 already established the repo's writer-aware display pattern (`Show` → `ShowTo(os.Stdout)` → `ShowRangeTo(w, ...)`, helpers take `w io.Writer` first). This change applies the identical transform to the one method family M20 skipped. Deliberately no design novelty: same naming, same delegation shape, same doc style, same test style.

## Decisions

1. **`To` variants stay out of `interfaces.go`.** M20's `ShowTo`/`ShowRangeTo`/`SummaryTo` are not part of `IDataList`/`IDataTable`; adding only the Types variants would make the interface asymmetric. Widening the interfaces for the whole family is a separate decision with downstream-implementor impact — out of scope.
2. **`printTypeRows` gains `w io.Writer` as its first parameter** (matching `printRowsColored`) instead of a package-level writer or method receiver — it is a stateless helper shared by paged output.
3. **Terminal width detection unchanged.** `ShowTypesRangeTo` keeps using `utils.GetTerminalWidth()` / `getDataListTerminalWidth()` (which probe `os.Stdout`) even when writing to a buffer — identical to the M20 behavior of `ShowRangeTo`, so rendered bytes match the stdout path. Width-aware rendering of non-terminal writers is explicitly not a goal.
4. **DataList nil-receiver guard writes to `w`.** `(*DataList) ShowTypesRangeTo` keeps the existing nil check but emits the error line to `w`, so a nil DataList in the REPL surfaces in the captured output rather than escaping to stdout.

## Failure modes considered

- **Byte-drift between old and new paths**: avoided by making `ShowTypes`/`ShowTypesRange` pure delegates (single rendering body, no duplicated format strings).
- **Missed print site inside the moved bodies**: every `fmt.Print*` inside the two `ShowTypesRange` bodies and `printTypeRows` must become `fmt.Fprint*(w, ...)`; verified by compiling with the helper renamed (signature change forces every call site through review) and by the ctx.Output regression test.
- **Concurrent AtomicDo semantics**: unchanged — the writer swap does not move any code in or out of the `AtomicDo` callback; the trailing blank-line print stays outside, as in `ShowRangeTo`.
