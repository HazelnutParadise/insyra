# Design: bump-vulnerable-deps

## Context

Four indirect dependencies carry advisories. Only thrift is on a code path insyra executes (Parquet footer decoding through `arrow/go/v17/parquet/file`). The repo promises a minimum Go of 1.25.x to downstream users, and `go get` will silently rewrite the `go` directive when a dependency's own directive is newer.

## Decisions

1. **Bump each module explicitly, one `go get` with four pinned versions** rather than `go get -u`, so nothing else moves. Verified on 2026-09-06: thrift v0.24.0, grpc v1.83.1, x/image v0.45.0 and x/crypto v0.55.0 each leave the directive at `go 1.25.12`; x/crypto v0.56.0 raises it to 1.26.0. *Alternative*: take v0.56.0 and raise the minimum Go — rejected, that is a separate decision with its own follow-up (chromedp chain).
2. **Guard the directive in the task list**: after `go mod tidy`, the change is only complete if `grep '^go ' go.mod` still prints `go 1.25.12`.
3. **No test changes.** The bump is verified by the existing `parquet`, `gplot`, root and `csvxl` tests plus `go build ./...` and `govulncheck ./...`; there is no insyra behaviour to pin.
4. **No changelog entry**: AGENTS.md exempts dependency bumps with no behavioural effect.

## Risks / Trade-offs

- [x/crypto v0.55.0 leaves GO-2026-6355/6354 (ssh) open] → ssh is not in the build graph; tracked in #203 until the Go-1.26 decision.
- [arrow's thrift-generated code compiled against 0.23 may not build against 0.24] → verified: `go build ./...` and `go test ./parquet/` pass on the bumped graph.
