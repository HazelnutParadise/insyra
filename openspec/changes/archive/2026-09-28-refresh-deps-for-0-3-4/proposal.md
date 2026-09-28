# Proposal: refresh-deps-for-0-3-4

## Why

`dev` is about to be merged into `main` as Huashan v0.3.4, and the `dependency-vulnerability-floor` spec requires a dependency refresh to land on `dev` first, as its own change.

## What Changes

- `go get -u -t ./... go@1.25.12`, then `go mod tidy`. One direct requirement moves, `go-echarts/go-echarts/v2` v2.7.2 → v2.7.3, which only adds a `TextStyle` option. Six indirect ones move: `andybalholm/brotli` v1.2.5, `google/pprof`, `klauspost/compress` v1.20.1, `pierrec/lz4/v4` v4.1.31, `richardlehane/mscfb` v1.0.9 and `google.golang.org/grpc` v1.84.0.
- grpc is no longer held at v1.83.2. GitHub's advisory GHSA-2v4p-qf9q-27wj now gives the v1.84 line its own fix, commit `d5a41119`, which v1.84.0 contains, so v1.84.0 is outside every range. The Go vulnerability database's GO-2026-6443 has not been updated since before v1.84.0 was tagged and still covers it; govulncheck lists it among the packages insyra imports but does not call, and does not fail. The `gRPC is past the xDS header advisory` requirement said the advisory covers v1.84.0; that sentence is corrected.
- `chromedp` stays at v0.12.1 and `cdproto` at the version it needs. `go get -u` moved them to v0.14.2, which requires `go-json-experiment/json`, and a govulncheck built with Go 1.25 panics on that package.
- `goccy/go-json` stays at v0.10.6. Its v0.11.0 and v0.11.1, released on 2026-09-27, rewrite the encoder and decoder to behave like `encoding/json`, and that changes what insyra returns: `ToJSON` writes `1e-7` for `1e-07`, and `ReadJSON` rejects `{"a":01}`, reads `1e400` as a string instead of failing, and replaces invalid UTF-8. A behaviour change belongs in a change of its own with its changelog entry, not in a refresh. The refresh requirement gains that rule as a scenario, and `AGENTS.md` states it.
- 28 modules, chromedp and cdproto among them, stay behind because their newest version declares `go 1.26` or later. They, the chromedp limit and go-json are listed in `AGENTS.md`'s Follow-ups.
- No behaviour change, so no changelog entry.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dependency-vulnerability-floor`: the gRPC requirement no longer claims the advisory covers v1.84.0, and says which database is the authority; the refresh requirement adds the rule that a bump changing insyra's results is its own change.

## Impact

- `go.mod`, `go.sum`, `AGENTS.md`, `openspec/specs/dependency-vulnerability-floor/spec.md`.
- Lands on `dev` only. `0.4` is on Go 1.26 and refreshes against its own directive.
