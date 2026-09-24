# Proposal: release-gates

## Why

Huashan v0.3.3 went out with the Vulnerability Scan job red on `main`, and four OpenSpec changes whose work it shipped were archived only after the release. The owner ruled on 2026-09-24 that a release needs every check green on `main`, and that changes are archived before the release, not after it.

## What Changes

- `AGENTS.md`: two release-time rules next to the existing ones. Every change whose work is in the release is archived on `dev` before the release branch is cut, and a release needs every CI check green on the release PR and on the resulting `main` merge commit.
- A new `release-gates` spec states both so they can be checked.

## Capabilities

### New Capabilities

- `release-gates`: what has to be true before a release is tagged.

### Modified Capabilities

(none)

## Impact

- `AGENTS.md` and the new spec. No code, nothing user-visible, no changelog entry.
- The next release waits on the Vulnerability Scan job, which fails on every branch because of GO-2026-6452. `AGENTS.md`'s Follow-ups records the choice that unblocks it.
