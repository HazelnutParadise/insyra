## Why

`gplot` draws every series from the shared palette with no way to change it, where `plot`'s bar, line and scatter charts have `Colors`. An "Other" series can only be gray when it happens to be the ninth, so five series plus "Other" draw "Other" in pink.

## What Changes

- `gplot`'s `BarChartConfig`, `LineChartConfig`, `StepChartConfig` and `ScatterChartConfig` gain `Colors []string`, replacing the palette. A short list repeats, as in `plot`.
- A value is `#RGB`, `#RRGGBB`, `#RRGGBBAA` or a CSS colour name in any case. An unreadable value makes the constructor return a nil chart and an error naming it.

## Impact

- `gplot/`, `go.mod` (`golang.org/x/image` becomes a direct dependency at the same version), `Docs/gplot.md`, both changelogs, `chart-constructors` spec.
