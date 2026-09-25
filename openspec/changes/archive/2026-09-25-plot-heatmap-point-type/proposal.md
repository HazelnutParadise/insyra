## Why

The fourth batch of #213 (PL-4). A heat map's points were of the unexported type `heatMapPoint[X, Y]` under an unexported constraint misspelled `heapMapAxisValue`, so a caller building points in a loop could not name the slice to collect them in. The owner chose on 2026-09-26 to export the type under the plain name and give the constructors the Go `New` prefix. `SaveHTML`'s extra flag was handled in the first batch; chart widths and heights stay CSS strings so `"100%"` keeps working.

## What Changes

- **BREAKING**: `HeatMapPoint[X, Y]` is the exported point type and `HeatMapAxis` its exported constraint. `HeatMapPoint(x, y, v)` becomes `NewHeatMapPoint(x, y, v)` and `HeatMapMissingPoint(x, y)` becomes `NewHeatMapMissingPoint(x, y)`; `CreateHeatMap` takes `...HeatMapPoint[X, Y]`.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `optional-values`: the heat map's points have a type a caller can name.

## Impact

- `plot/heatmap.go`, `plot/charts_test.go`, `plot/heatmap_point_test.go` (new); `Docs/plot.md`, `skills/insyra/references/plotting.md`, both changelogs.
