# Tasks

- [x] Tests first: segmented StochRSI and CR equal the separate segments for every setting, nothing crosses a start, the zero value is bit-identical, bad starts are refused, and a flat start no longer leaves StochRSI NaN (`quant/ta/oscillator_segment_test.go`).
- [x] `CROptions.SegmentStarts`, `KDOptions.SegmentStarts` for `StochRSI`, and the recursive smoothing that skips undefined raw values.
- [x] `Docs/quant-ta.md` states how to get TA-Lib's and TradingView's StochRSI, and entries in both CHANGELOGs.
- [x] `gofmt`, `golangci-lint run ./quant/...`, `go test ./quant/... -race`, `openspec validate --specs --strict` pass.
