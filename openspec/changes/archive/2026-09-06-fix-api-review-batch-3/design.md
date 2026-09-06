# Design: fix-api-review-batch-3

## Context

Hygiene items from the review that share one property: the current behaviour is wrong or unsafe, the fix stays inside the existing signature, and no other package depends on the wrong behaviour. Items that would remove a feature (lp's auto-install, the Google Maps crawler, `LogFatal`) or change a return shape stay out, per the user's instruction.

## Decisions

1. **SavePNG default** — `doesUseOnlineServiceOnFail` becomes `len(args) > 0 && args[0]`. The CLI `plot` command keeps the library default (off) and its error text tells the user how to opt in. *Alternative*: keep the default and log louder — rejected, a warning does not undo an upload.
2. **Error hook** — `pushError` sends to a buffered channel (capacity 1024) drained by a single goroutine started on first use; when full, the error is still recorded in the ring and the hook call is dropped with a once-per-process warning. Ordering is preserved. *Alternative*: call the hook synchronously — rejected because a hook that calls back into Insyra under the error mutex would deadlock.
3. **Config atomics** — `logLevel` becomes `atomic.Int32`, `coloredOutput` and `dontPanic` `atomic.Bool`; `defaultErrHandlingFunc` an `atomic.Pointer`. Getters and `colorText` read through them.
4. **Banner** — removed from `init.go`; `repl.Start` prints the same line once at startup so the interactive experience is unchanged and library users get silence.
5. **DetectEncoding** — after reading the sample, if `utf8.Valid` fails, trim trailing bytes that form an incomplete rune (walk back up to 3 bytes) and re-validate; only then fall through to chardet.
6. **DataList cleanup** — `ClearNaNs`/`ClearNils`/`ClearNilsAndNaNs` build a kept slice in one pass. `DropAll`/`ClearStrings` become plain loops. `IsEqualTo` compares with an `equalCell` helper that treats two float64 NaNs as equal. `IsTheSameAs` inherits it.
7. **Find hygiene** — an internal `findFirstIndex(value) (int, bool)` without warnings serves `FindColsIfContains*`; `containsSubstring` is replaced by `strings.Contains`; `Count` sums per-column counts in a loop; `Clone` clones columns in a loop. `AppendRowsByColName` sorts the map keys before adding columns (found by `TestReadJSONFileMatchesReadJSON` flaking on column order); existing columns keep their position, so only the order of columns first seen in the same row changes. *Alternative*: preserve JSON source order — rejected, `encoding/json` decodes objects into maps and the order is already gone.
8. **stats type guard** — `asDataList(dl insyra.IDataList) *insyra.DataList` returns the concrete list when it is one, otherwise `insyra.NewDataList(dl.Data()...)`; nil yields an empty list so the existing length checks produce their existing errors. Applied to the 13 assertion sites.
9. **parquet.Write** — same temp-file-then-rename shape as `ApplyCCL`; close errors through `insyra.LogWarning("parquet", …)`.
10. **Registry** — a package-level `sync.RWMutex` around `Register`, `Dispatch`, and `BuildCobraCommands`'s read.

## Risks / Trade-offs

- [A CLI user without local Chrome loses PNG export until they opt in] → the error text names the flag; documented in `Docs/cli-dsl.md`.
- [A hook that blocks stalls all later hook calls (single worker)] → same as any async logger; the ring buffer is unaffected, and the bounded channel drops rather than grows.
- [`IsEqualTo` changing for NaN] → previously `dl.IsEqualTo(dl.Clone())` was false for any list with NaN, which no caller could have wanted.
