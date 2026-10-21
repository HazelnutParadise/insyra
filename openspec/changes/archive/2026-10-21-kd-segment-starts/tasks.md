# Tasks

- [x] Tests first: segmented KD equals KD of each segment alone for both methods and several n, a segment shorter than the warm-up, the zero value bit-identical, bad starts refused (`quant/ta/kd_segment_test.go`).
- [x] `KDOptions.SegmentStarts`, `kdSegment`, and the `cleanSegmentStarts` and `segmentBounds` helpers.
- [x] `Docs/quant-ta.md` and entries in both CHANGELOGs.
- [x] `gofmt`, `golangci-lint run ./quant/...`, `go test ./quant/... -race`, `openspec validate --specs --strict` pass.
