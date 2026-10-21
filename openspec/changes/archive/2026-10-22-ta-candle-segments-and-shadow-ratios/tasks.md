# Tasks

- [x] Tests first: segmented patterns equal the separate segments, no pattern reads across a start, the zero value is bit-identical, bad starts are refused, and the two shadow ratios act separately (`quant/ta/candles_segment_test.go`).
- [x] `CandleOptions.SegmentStarts`, and `LongShadowRatio` and `ShortShadowRatio` in place of `ShadowRatio`; the flags for default options match the output measured before the change on a seeded 3,000-bar series.
- [x] `Docs/quant-ta.md` and an entry in both CHANGELOGs.
- [x] `gofmt`, `golangci-lint run ./quant/...`, `go test ./quant/... -race`, `openspec validate --specs --strict` pass.
