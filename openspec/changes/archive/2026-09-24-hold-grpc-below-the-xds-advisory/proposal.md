# Proposal: hold-grpc-below-the-xds-advisory

## Why

The refresh before Huashan v0.3.3 (`refresh-deps-for-0-3-3`) moved `google.golang.org/grpc` from v1.83.2 to v1.84.0, and v1.84.0 is inside GHSA-2v4p-qf9q-27wj, the xDS server crash the floor was written for. The advisory is fixed on the 1.83 line in v1.83.2, and on the newer line only in a `1.85.0-dev` pseudo-version; no 1.84 patch release exists. Dependabot raised alert 37 against `main` as soon as v0.3.3 was merged.

Two things let it through. The floor requirement says "v1.83.2 or later", and v1.84.0 is later. And the refresh was checked with `govulncheck` alone, whose database does not list the 1.84 range. The requirement also cites GHSA-mm6q-rjw5-hqhw, an ID GitHub does not know; the advisory is GHSA-2v4p-qf9q-27wj.

## What Changes

- `google.golang.org/grpc` v1.84.0 → v1.83.2. Nothing else in the module graph requires 1.84, so no other module moves.
- The gRPC floor names the right advisory and requires a version outside every one of its vulnerable ranges, not merely a later one.
- The refresh requirement adds a second check: every module in `go.mod` is compared against GitHub's advisory database, and a module whose newest version is inside an advisory range stops at the newest version outside it and is recorded like any other held-back module.
- `AGENTS.md`: the refresh rule gains the advisory check, grpc joins the held-back list, and the excelize follow-up records that GO-2026-6452 is a database error, not an open vulnerability.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dependency-vulnerability-floor`: the gRPC floor excludes every vulnerable range, and the refresh checks GitHub's advisory database.

## Impact

- `go.mod`, `go.sum`, `AGENTS.md`. No source change and nothing a user of the library sees, so no changelog entry.
- Lands on `dev` for v0.3.4. v0.3.3 as released requires grpc v1.84.0; grpc is indirect and insyra imports no gRPC package, but a downstream module cannot resolve below insyra's requirement, so v0.3.4 is the fix for them.
