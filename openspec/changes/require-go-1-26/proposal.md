# Proposal: require-go-1-26

## Why

Go 1.25 stopped receiving fixes when Go 1.27 was released, and go1.25.14 is its last release. On 2026-10-08 the Go vulnerability database published eleven standard-library advisories fixed only in go1.26.9 and go1.27.2, among them GO-2026-6599 and GO-2026-6600 in `html/template` and five HTTP/2 advisories in `net/http`. Built with go1.25.14, `govulncheck ./...` reports all eleven as reachable from insyra code, through `plot.SavePNG`, `DataTable.ToSQL` and `lpgen` among others, so the Vulnerability Scan on `dev` is red. The five HTTP/2 advisories are also published against `golang.org/x/net` v0.58.0, and every `x/net` from v0.59.0 declares `go 1.26.0`, so the `dependency-vulnerability-floor` spec stops that bump at the Go 1.25 directive.

AGENTS.md keeps the minimum Go version a separate, explicit decision. The owner decided on 2026-10-10 to move `dev` (the 0.3.x line) to Go 1.26, recorded on [#203](https://github.com/HazelnutParadise/insyra/issues/203). This change is that decision on its own, so the dependency bumps it unblocks land as their own changes.

The newest 1.26 release on go.dev is 1.26.9 (checked 2026-10-10), and it is the first 1.26 release outside all eleven advisories.

## What Changes

- `go.mod`'s `go` directive becomes `1.26.9`. `go mod tidy` changes nothing else.
- The govulncheck workflow scans with the latest 1.26 patch and sets `check-latest: true`, so setup-go fetches that patch instead of taking the runner's cached one.
- `Docs/README.md` and the tutorials say Go 1.26+.
- Both changelogs carry a BREAKING entry.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dependency-vulnerability-floor`: adds a requirement that the minimum Go version moves only as its own, explicitly decided change, and records that `dev` is on `1.26.9`. The existing requirement that dependency bumps keep the directive is unchanged.

## Impact

- **BREAKING**: building Insyra needs Go 1.26 or newer. With Go 1.21+ the `go` command downloads the 1.26.9 toolchain automatically unless `GOTOOLCHAIN=local` is set.
- No dependency version changes in this change. The scan stays red on the five `x/net` advisories until `bump-x-net-for-http2-advisories` lands.
