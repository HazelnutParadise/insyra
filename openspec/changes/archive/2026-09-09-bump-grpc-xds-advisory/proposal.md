# Proposal: bump-grpc-xds-advisory

## Why

Dependabot alert 36 (High) reports `google.golang.org/grpc` v1.83.1 against GHSA-mm6q-rjw5-hqhw: a gRPC-Go xDS server crashes on a request with no `:authority` and no `Host` header. The alert is on the default branch, so it describes the released v0.3.2.

Insyra imports no gRPC package — it arrives through the Arrow/Parquet chain — so the vulnerable code is not reachable from insyra. The bump is still worth taking on its own: it is one patch version, it costs nothing, and leaving a High alert standing on the default branch makes the next real one easier to miss.

## What Changes

- `google.golang.org/grpc` v1.83.1 → v1.83.2, the first patched version.
- `golang.org/x/net` v0.57.0 → v0.58.0, pulled in by that bump.
- The `go` directive stays at 1.25.12: grpc v1.83.2 declares `go 1.25.0`, so the minimum-Go promise is unaffected.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dependency-vulnerability-floor`: the version floor gains gRPC.

## Impact

- `go.mod`, `go.sum`. No source change, nothing user-visible, so no changelog entry.
- Lands on `dev` (the 0.3.x line, which is what ships next) and is merged into `0.4`.
