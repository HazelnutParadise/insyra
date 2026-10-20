# Tasks

- [x] Split `BlockBootstrap`'s resampling into index drawing and value reading; existing seeded tests unchanged and passing (`quant/bootstrap.go`).
- [x] `DataSnoopingOptions`, `ResamplingScheme` with `String()`, `RealityCheck`, `SPATest` (`quant/data_snooping.go`, `quant/enum_names.go`).
- [x] Python reference of the paper's formulas for dbar, ω and T_SPA (`quant/testdata/data_snooping_reference.py`, `quant/testdata/data_snooping_baseline.json`) and a test against it.
- [x] Statistical tests with fixed seeds (null data, a planted strong strategy, a strong strategy among poor ones), joint-index and reproducibility tests, bad-input tests.
- [x] `Docs/quant.md` section, including how the results differ from Python `arch`, and entries in both CHANGELOGs.
- [x] `gofmt`, `golangci-lint run`, `go test ./quant/...` and `openspec validate --specs --strict` pass.
