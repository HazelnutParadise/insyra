## Why

A trailing `...T` or `...XxxOptions` parameter stands for one optional value, but most functions that take one kept the first value and dropped the rest without a word, `finance` kept the last, and the `nn` pooling and convolution constructors threw every option away in favour of the defaults. A caller who passed two option sets never learned that one was ignored. The owner ruled on 2026-09-25 (#213) that more than one value is always an error, reported through the function's normal error path; this is the first batch of #213 and applies that rule everywhere the audit found it broken.

## What Changes

- **BREAKING (behaviour, not signature)**: a second value becomes an error in:
  - `DataList.Shift` / `DataTable.ShiftCol`, `FillForward`, `FillBackward`, `FillByInterpolation`, `Rank` (which recorded an error and ranked anyway), `Sample`, `SampleFrac`, `Shuffle`, `TrainTestSplit`, `Describe` (list, table, grouped). Chainable methods record the error and change nothing.
  - `ReadSQL`, `ReadSQLContext`, `ReadSQLStream`, `ToSQL`, `ToSQLContext`, `ReadCSV_File`, `plot.SaveHTML`, and every `finance` function taking `opts ...Options`: they return the error.
  - `nn.NewTape`: reported by `Param` and `Backward`, the calls every training step makes, because `NewTape` has no error result. `Conv2D`, `MaxPool2D`, `AvgPool2D`: reported by `Build`, where the layers report every other invalid setting.
  - `accel.NewSession`: reported by `Discover`.
  - `datafetch` `GetReviews`: logged, and nil is returned before any request.
- The functions that already refused a second value (`Sort`, the `stats` and `ml` option structs, the `nn` kernels, `csvxl`'s encodings, `SavePNG`) are unchanged.
- `Show` / `ShowRange`, whose `startEnd ...any` is a positional pair rather than one optional value, belong to the batch that reshapes signatures, not this one.

## Capabilities

### New Capabilities
- `optional-values`: a trailing optional parameter takes at most one value.

### Modified Capabilities
None.

## Impact

- Root: `optional_values.go` (new), `datalist.go`, `datalist_window.go`, `datalist_impute.go`, `datalist_sampling.go`, `datatable_sampling.go`, `describe_options.go`, `datatable_from_sql.go`, `datatable_to_sql.go`, `read.go`.
- `finance/*`, `nn/autodiff.go`, `nn/layers_catalog.go`, `accel/session.go`, `accel/discovery.go`, `plot/save_chart.go`, `datafetch/googleMapsCommentCrawler.go`.
- `Docs/DataList.md`, `Docs/DataTable.md`, `Docs/finance.md`, `Docs/nn.md`, `Docs/accel.md`, `Docs/plot.md`; `skills/insyra/SKILL.md`; both changelogs.
