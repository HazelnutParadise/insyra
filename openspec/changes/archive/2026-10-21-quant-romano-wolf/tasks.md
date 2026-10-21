# Tasks

- [x] Tests first: a Python reference of the stepdown on supplied bootstrap positions (`quant/testdata/romano_wolf_baseline.py`, `.json`) and the Go comparison, a planted-strong-strategy table, a familywise-error check over null tables, bad-input tests, Alpha monotonicity.
- [x] `RomanoWolfOptions`, `RomanoWolfResult`, `RomanoWolf` and the FFT long-run variance (`quant/romano_wolf.go`); the `DataSnoopingOptions` comment names the third user.
- [x] `Docs/quant.md` section and entries in both CHANGELOGs.
- [x] `gofmt`, `golangci-lint run ./quant/...`, `go test ./quant/... -race`, `openspec validate --specs --strict` pass; timing of 5000 replications at T=2500, N=100 recorded in the pull request.
