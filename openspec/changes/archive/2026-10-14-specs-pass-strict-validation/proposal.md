# Proposal: specs-pass-strict-validation

## Why

Two `AGENTS.md` follow-ups recorded that the main specs fail `openspec validate --specs --strict`: 46 of 105 on 2026-09-12, 36 of 136 on 2026-09-28. Nothing in CI ran the command, so the count kept climbing. Measured on `0.4` at 8f1f6396 with OpenSpec 1.14.1, 78 of the 173 main specs failed:

- 35 had a Purpose that `openspec archive` writes for a new capability (`TBD - created by archiving change <id>. Update Purpose after archive.`) or one shorter than 50 characters.
- 49 held 91 requirements whose text before the first scenario is longer than 500 characters, the longest 2,154.

The owner asked for the follow-up to be handled on 2026-10-14.

## What Changes

- Every Purpose is written from the requirements under it: one or two sentences on what the capability is for. A short Chinese Purpose is expanded in Chinese.
- Every long requirement is split into requirements that each state one behaviour. The first keeps the original header, so a `MODIFIED` delta written against the old header still applies. Every rule and every scenario of the original survives. A scenario that covered two rules is split between them, and a part that had no scenario of its own gets one that restates a rule already in the text. Each such scenario was checked against the code, `go doc`, the docs or the workflow before it was written. A few requirements are shortened instead, by tightening wording without dropping a rule.
- The main specs are edited directly rather than through deltas: a delta cannot carry a Purpose for a capability that already exists, and 91 `MODIFIED` plus `ADDED` blocks would repeat every requirement. Nothing a requirement asks of the code changes.
- The GolangCI-Lint workflow gains an `OpenSpec strict validation` job that runs `openspec validate --specs --strict` with OpenSpec 1.14.1 on every push and pull request to `main`, `dev` and `0.4`.
- `AGENTS.md`: the archive step says to replace the placeholder Purpose in the same commit, a paragraph states the strict-validation rule and how to split a requirement, and the two follow-ups are removed.

## Capabilities

### New Capabilities

- `spec-validation`: every main spec passes strict validation, CI enforces it, and a split keeps a requirement's meaning.

### Modified Capabilities

None. The 78 edited main specs change in wording and structure only.

## Impact

- `openspec/specs/*/spec.md` (78 specs), `.github/workflows/golangci-lint.yml`, `AGENTS.md`, `delivery-status.md`.
- No library, CLI or documentation behaviour changes, so there is no changelog entry.
- A future archive that leaves a placeholder Purpose, or writes a requirement over 500 characters, fails CI on that commit.
