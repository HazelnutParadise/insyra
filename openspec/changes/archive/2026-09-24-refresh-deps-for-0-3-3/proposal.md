# Proposal: refresh-deps-for-0-3-3

## Why

`dev` is about to be merged into `main` as Huashan v0.3.3, and the `dependency-vulnerability-floor` spec requires a dependency refresh to land on `dev` first, as its own change.

## What Changes

- Every module moves to the newest version that keeps the `go 1.25.12` directive: `go get -u -t ./... go@1.25.12`, then `go mod tidy`. Ten direct requirements move, among them `gogpu/wgpu` v0.30.35 → v0.34.5, `gogpu/gputypes` v0.5.1 → v0.8.0, `apoplexi24/gpandas` v0.2.0 → v0.5.0, `wnjoon/go-yfinance` v1.5.1 → v1.7.0 and `modernc.org/sqlite` v1.53.0 → v1.59.0, plus 44 indirect ones.
- `gogpu/wgpu` v0.34.1 removed its `BufferUsage` and `BindGroupLayoutEntry` aliases of the `gputypes` types. The four places in `accel/internal/wgpu` that named them now name the `gputypes` types directly. An alias is the same type, so nothing the code does changes.
- `chromedp` moves from v0.11.2 to v0.12.1 and stops there. From v0.13.0 it requires `go-json-experiment/json`, and a govulncheck built with Go 1.25, which is what the Vulnerability Scan job runs, panics on that package.
- 22 modules stay behind because their newest version declares `go 1.26`. They and the chromedp limit are listed in `AGENTS.md`'s Follow-ups.
- No behaviour change, so no changelog entry.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

(none — this carries out the existing "Dependencies are refreshed before every release" requirement)

## Impact

- `go.mod`, `go.sum`, `accel/internal/wgpu/matmul.go`, `accel/internal/wgpu/wgpu.go`, `AGENTS.md`.
- Lands on `dev` only. `0.4` is on Go 1.26 and refreshes against its own directive.
