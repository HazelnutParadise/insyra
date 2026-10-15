# Proposal: specs-pass-strict-validation

## Why

The `AGENTS.md` follow-up of 2026-09-14 recorded 26 of 101 main specs failing `openspec validate --specs --strict` on `dev`. Nothing in CI ran the command. Measured on `dev` at 341bbf51 with OpenSpec 1.14.1, 40 of the 143 main specs failed:

- 26 had a Purpose that `openspec archive` writes for a new capability (`TBD - created by archiving change <id>. Update Purpose after archive.`) or one shorter than 50 characters.
- 17 held 27 requirements whose text before the first scenario is longer than 500 characters.

`0.4` fixed the same problem on 2026-10-14 in a change of the same name (aa2a9325). The owner asked for this line to get the same treatment on 2026-10-15.

## What Changes

- 27 of the failing specs are byte for byte what `0.4` had before its fix, and take `0.4`'s fixed text unchanged, so the next merge of `dev` into `0.4` meets the same text on both sides.
- The other 13 differ from `0.4` and are fixed on this line. Each Purpose says what that spec covers here: where a spec lacks requirements `0.4` has (aliases in `command-registry`, `exit` in `script-runner`, own-type fit results in `ml-protocol`), its Purpose leaves them out. A long requirement whose text matches `0.4`'s is split as `0.4` split it. The other four are split by hand under the same rule: the first part keeps the original header, and every rule and scenario survives. One part had no scenario of its own and got one that restates its rule, checked against the code (`EWMCol` reads a column by name like `GetColByName`).
- The main specs are edited directly rather than through deltas, for the reason `0.4`'s change gives: a delta cannot carry a Purpose for a capability that already exists. Nothing a requirement asks of the code changes.
- The GolangCI-Lint workflow gains the `OpenSpec strict validation` job `0.4` has, running OpenSpec 1.14.1 on every push and pull request to `main` and `dev`.
- `AGENTS.md`: the archive step and the strict-validation rule as on `0.4`; the follow-up is removed.

## Capabilities

### New Capabilities

- `spec-validation`: every main spec passes strict validation, CI enforces it, and a split keeps a requirement's meaning.

### Modified Capabilities

None. The 40 edited main specs change in wording and structure only.

## Impact

- `openspec/specs/*/spec.md` (40 specs), `.github/workflows/golangci-lint.yml`, `AGENTS.md`, `delivery-status.md`.
- No library, CLI or documentation behaviour changes, so there is no changelog entry.
- `openspec/specs/spec-validation/spec.md` names `main` and `dev` here and `main`, `dev` and `0.4` on `0.4`; the merge into `0.4` keeps `0.4`'s.
