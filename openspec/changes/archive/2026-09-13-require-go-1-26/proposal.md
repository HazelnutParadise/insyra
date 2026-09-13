# Proposal: require-go-1-26

## Why

The `0.4` series will make [go-milp](https://github.com/daniel-sullivan/go-milp) the default solver behind `lp` ([#257](https://github.com/HazelnutParadise/insyra/issues/257), [#372](https://github.com/HazelnutParadise/insyra/issues/372)). go-milp's `go.mod` declares `go 1.26`, so Insyra cannot depend on it while its own `go` directive is `1.25.12`. The owner decided on 2026-09-13 to raise the minimum Go version to 1.26 on `0.4` only; the 0.3.x line on `dev` stays on Go 1.25.

AGENTS.md keeps the minimum Go version a separate, explicit decision rather than a side effect of a dependency bump. This change is that decision, made on its own so the `lp` change that needs it does not also carry a toolchain move.

The newest 1.26 release on go.dev is 1.26.8 (checked 2026-09-13). Earlier 1.26 patches carry standard-library fixes that govulncheck would report against the toolchain itself, which is why the 1.25 line was pinned to its latest patch too.

## What Changes

- `go.mod`'s `go` directive becomes `1.26.8`.
- The govulncheck workflow scans with the latest 1.26 patch instead of 1.25.
- `Docs/README.md` and the tutorials say Go 1.26+.
- The chromedp follow-up in AGENTS.md records that its Go 1.26 precondition is now met on `0.4`; retrying the upgrade stays a separate task.
- Both changelogs carry a BREAKING entry.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dependency-vulnerability-floor`: adds a requirement that the minimum Go version moves only as its own, explicitly decided change, and records that the `0.4` line is on `1.26.8`. The existing requirement that dependency bumps keep the directive is unchanged.

## Impact

- **BREAKING**: building Insyra `0.4` needs Go 1.26 or newer. With Go 1.21+ the `go` command downloads the 1.26.8 toolchain automatically unless `GOTOOLCHAIN=local` is set.
- No dependency version changes in this change.
