# Proposal: bump-vulnerable-deps

## Why

Dependabot reports two high-severity alerts on the default branch and `govulncheck` lists two more module-level advisories. One of them is reachable: `apache/thrift v0.23.0` (CVE-2026-43871) decodes every Parquet footer insyra reads, so a crafted `.parquet` file can hang `parquet.Read`/`Inspect`/`Stream`/`ReadColumn`. The other three (`grpc`, `x/crypto`, `x/image`) are not reachable from insyra code but stay visible to every downstream scanner. Issues #201–#204 hold the analysis.

## What Changes

- `github.com/apache/thrift` v0.23.0 → v0.24.0 (#201, reachable, fixes CVE-2026-43871).
- `google.golang.org/grpc` v1.82.1 → v1.83.1 (#202, unreachable, clears CVE-2026-84304).
- `golang.org/x/image` v0.44.0 → v0.45.0 (#204, unreachable, clears GO-2026-6222).
- `golang.org/x/crypto` v0.54.0 → v0.55.0 (#203, unreachable, clears GO-2026-6303). v0.56.0 is **not** taken: it forces the `go` directive to 1.26.0, breaking the minimum-Go promise; it waits for the Go-1.26 decision already tracked in the chromedp follow-up.
- The `go 1.25.12` directive stays unchanged.
- No behaviour change, no changelog entry (dependency bump with no behavioural effect, per AGENTS.md).

## Capabilities

### New Capabilities

- `dependency-vulnerability-floor`: the module never requires a dependency version with a known reachable vulnerability, and dependency bumps never raise the module's minimum Go version as a side effect.

### Modified Capabilities

(none)

## Impact

- `go.mod`, `go.sum` only.
- Packages whose dependency graph changes: `parquet` (thrift, grpc), root and `csvxl` (excelize → x/crypto, x/image).
