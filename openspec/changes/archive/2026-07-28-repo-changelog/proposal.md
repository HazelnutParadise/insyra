# Proposal: repo-changelog

## Why

Pending release notes live in [GitHub Discussion #6](https://github.com/HazelnutParadise/insyra/discussions/6), outside the repository. Nobody editing code passes through that discussion, so entries are written from memory before a release instead of alongside the change that caused them — and the discussion cannot be reviewed, diffed, or required as part of a pull request. Moving the pending notes into the repo puts the description of a change in the same pull request as the change itself.

## What Changes

- Add `CHANGELOG.md` (English) and `CHANGELOG_TW.md` (Traditional Chinese) at the repository root, mirroring the existing `README.md` / `README_TW.md` pairing.
- Both files open with a single `## Unreleased` section and carry no backfilled history: releases up to and including v0.3.0 stay in [GitHub Releases](https://github.com/HazelnutParadise/insyra/releases), which each file links to.
- Entries are grouped by package exactly as release notes are today (`### Core`, `### CLI`, `` ### `stats` ``, …), one heading level deeper so that promoting `###` to `##` at release time yields the release note verbatim.
- Seed `## Unreleased` with the one user-visible change landed since v0.3.0: `CSVReadOptions.RawStrings` ([#189](https://github.com/HazelnutParadise/insyra/pull/189)).
- `AGENTS.md` gains a changelog rule inside the existing "Docs & Skills Must Stay in Sync" policy: a user-visible change updates both changelog files in the same change, and the OpenSpec workflow treats that update as part of the change rather than a follow-up.
- `CONTRIBUTING.md` gains the same rule phrased for human contributors, including what does *not* warrant an entry.
- `README.md`, `README_TW.md`, and `Docs/README.md` link to the changelog. `Docs/` gets no mirror copy — docsify's basePath is `/Docs`, so a repo-root file cannot be served from there, and a duplicate would drift.

Explicitly not included:

- No CI check that fails a pull request touching `.go` without touching the changelog. Pure refactors, test fixes, and dependency bumps would be blocked wrongly, and the rule in `AGENTS.md` already binds the agents that do most of the work here.
- No release-time conversion script. Promoting two heading levels by hand takes seconds; a script is worth writing only once that proves annoying.

## Capabilities

### New Capabilities
- `changelog`: The repository-hosted changelog — its two language files, section structure, what qualifies as an entry, and when contributors must update it.

### Modified Capabilities

None. The existing specs (`cli-entry`, `command-registry`, `dsl-commands`, `env-management`, `repl-engine`, `script-runner`) all describe CLI behavior and are untouched by this change.

## Impact

- New: `CHANGELOG.md`, `CHANGELOG_TW.md`
- Modified: `AGENTS.md`, `CONTRIBUTING.md`, `README.md`, `README_TW.md`, `Docs/README.md`
- No Go code, tests, dependencies, or CI workflows change.
- Outside the repo, and outside this change's file edits: Discussion #6 needs a comment pointing readers at the changelog. That is a maintainer action requiring explicit approval, tracked in `tasks.md` as a manual step.
