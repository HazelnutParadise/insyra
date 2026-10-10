# Proposal: take-deps-held-by-go-1-25

## Why

Until `require-go-1-26`, the 1.25 directive held a group of modules below their newest versions, because each newest version declares `go 1.26` or later. AGENTS.md recorded them in a follow-up, together with `chromedp`, which was held at v0.12.1 because govulncheck built with Go 1.25 panics on the `go-json-experiment/json` package that later `chromedp` versions require. With the directive at 1.26.9 and the Vulnerability Scan built with Go 1.26, both reasons are gone, and the owner asked on [#203](https://github.com/HazelnutParadise/insyra/issues/203) for these modules to move with the Go 1.26 decision.

## What Changes

Every module in `go.mod` whose newest version declares `go 1.26`, plus the `chromedp` chain, moves to its newest version that keeps the `go` directive at 1.26.9, with the modules those versions require:

- `golang.org/x/crypto` v0.58.0, `exp`, `image` v0.47.0, `mod` v0.42.0, `net` v0.61.0, `oauth2` v0.37.0, `sync` v0.24.0, `sys` v0.49.0, `telemetry`, `term` v0.47.0, `text` v0.43.0, `time` v0.16.0, `tools` v0.51.0.
- `google.golang.org/api` v0.301.0, `google.golang.org/genproto` with `googleapis/api` and `googleapis/rpc`, `cloud.google.com/go/auth` v0.24.1, `auth/oauth2adapt` v0.3.0, `bigquery` v1.85.0, `compute/metadata` v0.10.0, `iam` v1.14.0, `github.com/googleapis/gax-go/v2` v2.26.2, `github.com/google/s2a-go` v0.1.11, `github.com/google/pprof`.
- `go.opentelemetry.io/otel`, `metric`, `trace` v1.47.0 (adding `otel/log`), and the `otelgrpc` and `otelhttp` instrumentation v0.72.0.
- `github.com/quic-go/quic-go` v0.63.0, `github.com/twpayne/go-geom` v1.7.0, `modernc.org/libc` v1.77.1, `modernc.org/sqlite` v1.60.1, `github.com/apache/thrift` v0.25.0, `github.com/Microsoft/go-winio` v0.6.3.
- `github.com/chromedp/chromedp` v0.12.1 → v0.16.0 with the `cdproto` it requires, adding `go-json-experiment/json`. Nothing newer works: v0.17.0 through v0.19.1 declare `go 1.27`, and v0.20 changes the API that `go-echarts/snapshot-chromedp` v0.0.5, its newest release and the renderer behind `plot.SavePNG`, calls, so the build fails. The held-back version is recorded in AGENTS.md.

Every moved module is outside every vulnerable range in GitHub's advisory database. Modules whose newest version still declares `go 1.25` or older and was released after the 2026-09-28 refresh are left to the next pre-release refresh.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dependency-vulnerability-floor`: the refresh requirement gains a scenario for a newest version that breaks the build.

## Impact

- `go.mod`, `go.sum`, AGENTS.md. No source change.
- Nothing a user of the library sees changes, so no changelog entry: a bar chart saved through `plot.SavePNG` and a Parquet file written through `parquet.Write` are byte-identical before and after.
