# Proposal: bump-x-net-for-http2-advisories

## Why

On 2026-10-09 the Go vulnerability database published five advisories against `golang.org/x/net` v0.58.0, the version this line requires, all fixed in v0.60.0: GO-2026-6617 (an HTTP/2 server crash from an HPACK encoder race), GO-2026-6612, GO-2026-6611 and GO-2026-6603 (HTTP/2 server flow-control, CPU and memory exhaustion), and GO-2026-6610 (the HTTP/2 transport accepting malformed framing headers). `govulncheck ./...` reports all five as reachable from insyra code through `net/http`'s bundled HTTP/2, for example from `datafetch` and `insyra.ReadJSON`, so the Vulnerability Scan failed on `0.4` at da65745e. The `dependency-vulnerability-floor` spec requires a reachable advisory to be fixed by moving the module to its first patched version in a change of its own.

## What Changes

- `golang.org/x/net` v0.58.0 → v0.60.0, the first patched version. It declares `go 1.26.0`, so the `go` directive stays at 1.26.8.
- v0.60.0 requires newer versions of its siblings, which move with it: `golang.org/x/crypto` v0.55.0 → v0.57.0, `x/sys` v0.47.0 → v0.48.0, `x/term` v0.45.0 → v0.46.0, `x/text` v0.41.0 → v0.42.0, and `go mod tidy` takes `x/mod` v0.40.0 → v0.41.0 and `x/sync` v0.22.0 → v0.23.0. Nothing else moves.
- Every module that moved was checked against GitHub's advisory database: none is inside an advisory range. GitHub had not yet listed the five `x/net` advisories, so `govulncheck` was the check that found them.
- The floor gains a requirement naming the `x/net` version.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dependency-vulnerability-floor`: `golang.org/x/net` is past the October 2026 HTTP/2 advisories.

## Impact

- `go.mod`, `go.sum`. No source change and nothing a user of the library sees, so no changelog entry.
- Verified with `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` and `govulncheck ./...` on go1.26.9 (0 reachable vulnerabilities).
