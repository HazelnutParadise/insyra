## Context

Release notes for Insyra are published on GitHub Releases in two languages, English first, then Traditional Chinese after a `---` separator, with entries grouped by package (`## Core`, `## CLI`, `` ## `stats` ``, …). Entries for the *next* release accumulate in [Discussion #6](https://github.com/HazelnutParadise/insyra/discussions/6), which uses a `v0.0.9+` naming convention for "the version after v0.0.9, number not yet decided".

The discussion is outside the repository. A pull request that changes behavior does not touch it, so an entry gets written later, from memory, by whoever prepares the release — and it can never be part of a code review. Sixty-seven releases exist; v0.3.0 (2026-07-18) is current, and one user-visible change has landed since it ([#189](https://github.com/HazelnutParadise/insyra/pull/189), `CSVReadOptions.RawStrings`).

Most work in this repository is driven by AI coding agents reading `AGENTS.md`, with human contributors following `CONTRIBUTING.md`.

## Goals / Non-Goals

**Goals:**

- A pending changelog entry lands in the same pull request as the change that caused it.
- Publishing a release means copying a section and promoting its headings — no rewriting.
- English and Traditional Chinese stay in step, matching how `README.md` / `README_TW.md` already work.
- The rule is discoverable by both audiences that do work here: agents via `AGENTS.md`, humans via `CONTRIBUTING.md`.

**Non-Goals:**

- Reproducing the sixty-six releases published before v0.3.0.
- Enforcing the rule mechanically in CI.
- Automating the release-time transformation.
- Changing how release notes themselves are published on GitHub.

## Decisions

### Two files, not one bilingual file

`CHANGELOG.md` and `CHANGELOG_TW.md`, matching `README.md` / `README_TW.md`.

*Alternative considered:* a single file with both languages inside each version section. That makes it nearly impossible to update one language and forget the other, since the two entries sit adjacent. It was rejected in favor of consistency with the existing README pairing and to keep each file readable in one language without scrolling past the other. The cost — an entry can be added to one file and missed in the other — is carried by an explicit synchronization rule in `AGENTS.md` and `CONTRIBUTING.md`, which is the same mechanism already governing the READMEs and `Docs/`.

### Version at `##`, package at `###`

The release note puts package headings at `##` because a release note has no version heading — the release title carries the version. A changelog needs the version inside the document, so everything shifts down one level.

This makes release-time transformation mechanical: take the version section body, replace `### ` with `## `, done. Nothing else about the text changes.

*Alternative considered:* version at `#`, package at `##`, matching the release note exactly with no transformation. Rejected because it leaves no room for a `# Changelog` document title and reads oddly for a file whose sections are versions.

### No backfill

Each file's header links to GitHub Releases for anything up to and including v0.3.0.

*Alternative considered:* backfilling v0.3.0 only, so the file ships with a worked example of the format. Rejected: the `## Unreleased` section already carries a real entry (#189), which serves the same purpose. *Alternative considered:* backfilling all sixty-seven releases. Rejected — early release notes are not grouped by package and would need substantial rewriting, and GitHub Releases already serves that history with search.

### No CI enforcement

A check that fails any pull request touching `.go` without touching the changelog would wrongly block refactors, test fixes, and dependency bumps. Adding an exemption mechanism to work around that is more machinery than the problem justifies for a repository this size.

The rule instead lives where the work actually starts: `AGENTS.md`, which agents read before every task, and `CONTRIBUTING.md`. If human contribution volume grows and entries start getting missed, this decision is worth revisiting.

### Docs site links out rather than mirroring

docsify serves `Docs/` as its basePath and cannot reach a repository-root file. Copying the changelog into `Docs/` would create a second file that drifts. `Docs/README.md` links to the changelog on GitHub instead.

## Risks / Trade-offs

- **An entry is added to `CHANGELOG.md` and missed in `CHANGELOG_TW.md`** → The rule in `AGENTS.md` and `CONTRIBUTING.md` names both files explicitly. At release time, a maintainer copying both sections notices a mismatched entry count immediately, so the error surfaces before publication rather than after.
- **Entries get written at commit granularity and need rewriting at release time** → Both policy documents state the unit is a user-visible change, not a commit, and give examples of what does not warrant an entry.
- **Concurrent pull requests conflict on the same changelog lines** → Entries append to the end of their package section rather than the top, which keeps conflicts to genuinely simultaneous edits of the same package. Low risk at current contribution volume.
- **`## Unreleased` is left un-renamed at release time** → The spec states the rename explicitly, and `CONTRIBUTING.md` documents the release step. Not automated; a missed rename shows up as a `## Unreleased` section sitting under a newer one.
- **Discussion #6 keeps accumulating entries in parallel** → A comment on the discussion points readers at the changelog. It is left unlocked, so this is mitigation rather than prevention; locking it is a maintainer decision, not part of this change.

## Migration Plan

1. Create both changelog files with a single `## Unreleased` section seeded from #189.
2. Update `AGENTS.md`, `CONTRIBUTING.md`, and the three entry-point documents in the same change.
3. Post the pointer comment on Discussion #6 — a manual maintainer action outside the repository, taken only after explicit approval.

There is nothing to roll back beyond deleting the files; no code, build, or release tooling depends on them.

## Open Questions

- Whether Discussion #6 should eventually be locked once the changelog is established. Deliberately left open — locking removes a place people already know to look, and the pointer comment may be sufficient.
