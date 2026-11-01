## Why

Setting a colour means knowing its hex value, so giving an "Other" series the palette's gray takes a lookup, and a copied value drifts when the palette changes. `gplot` also has no way to set a chart's background, so the brand's cream can be used only in `plot`.

## What Changes

- `plot` and `gplot` export `ColorBlue`, `ColorOrange`, `ColorRed`, `ColorTeal`, `ColorPurple`, `ColorPink`, `ColorGreen`, `ColorBrightBlue`, `ColorGray` and `ColorCream`, plain string constants holding the palette and the brand's background colour.
- Every `gplot` config gains `BackgroundColor`, read like a value of `Colors`; empty keeps white.

## Impact

- `plot/`, `gplot/`, `internal/palette/`, `Docs/plot.md`, `Docs/gplot.md`, both changelogs, `chart-constructors` spec.
