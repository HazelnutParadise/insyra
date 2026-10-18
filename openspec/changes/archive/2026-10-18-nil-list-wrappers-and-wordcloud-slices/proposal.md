# Proposal: nil-list-wrappers-and-wordcloud-slices

## Why

Two panics that break "the library does not panic by default":

- [#412](https://github.com/HazelnutParadise/insyra/issues/412): `plot`'s `isNilList` only knew a nil `*insyra.DataList`. A value embedding one, such as `isr.DL`, satisfies `IDataList`, passed the check and panicked on its first method call. `stats`' `asDataList` has the same gap.
- [#411](https://github.com/HazelnutParadise/insyra/issues/411): `plot.CreateWordCloud` counted cells in a `map[any]float32`, so a slice cell panicked with `hash of unhashable type`.

## What Changes

- New `insyra.IsNilList(dl IDataList) bool`: true for a nil interface, a nil `*DataList`, or a value embedding a nil `*DataList`. `plot` and `stats` use it instead of each keeping its own check.
- `CreateWordCloud` counts words by the text the chart shows (`conv.ToString`), as the issue recommends. Side effect: `1` and `"1"` are one word, which they already looked like on the chart.
- Not changed: `gplot` on this line has no nil-list check at all; `0.4` adds one with its constructors returning errors.

## Capabilities

### Modified Capabilities

- `error-philosophy`: a wrapper holding a nil list and a slice cell in a word cloud do not panic.

## Impact

`datalist_nil.go`, `plot/input.go`, `plot/wordcloud.go`, `stats/asdatalist.go`, tests, `Docs/DataList.md`, `Docs/plot.md`, both CHANGELOGs.
