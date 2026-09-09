# Proposal: refresh-deps-before-release

## Why

Dependency bumps happen when Dependabot files an alert, which means the module graph only moves when something is already broken. The grpc xDS advisory landed that way on 2026-09-09, one patch version behind, and the alert sat on the default branch because `main` only moves at a release.

Nothing in the operating contract says to look at dependencies on the way to a release, so nobody does. Two consequences: an advisory published between releases ships anyway, and a bump that has been available for months is taken under time pressure at the moment it becomes urgent, when it is least safe to take.

## What Changes

- **A dependency refresh becomes part of merging to `main`.** Before the merge, every direct and indirect dependency is moved to the newest version that leaves the `go` directive unchanged, in a change of its own, verified with the full suite and `govulncheck`.
- **What cannot move is written down.** A dependency that is held back — because its newest version needs a newer Go, or because it breaks a tool the CI depends on — goes into `AGENTS.md`'s Follow-ups with the reason, the way the chromedp chain already is.
- `AGENTS.md` gains the rule; the `dependency-vulnerability-floor` spec gains the requirement so it is checkable.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dependency-vulnerability-floor`: gains a requirement about when the refresh happens, alongside the existing ones about advisories and the minimum Go version.

## Impact

- `AGENTS.md`. No code, nothing user-visible, no changelog entry.
- Lands on `dev` and is merged into `0.4`, because the release goes out from `dev`.
