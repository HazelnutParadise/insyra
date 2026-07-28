# changelog Specification

## Purpose
TBD - created by archiving change repo-changelog. Update Purpose after archive.
## Requirements
### Requirement: Changelog files exist at the repository root

The repository SHALL host its changelog in two files at the root: `CHANGELOG.md` in English and `CHANGELOG_TW.md` in Traditional Chinese. The two files SHALL describe the same set of changes at all times.

#### Scenario: Both language files are present

- **WHEN** the repository is checked out at any commit on `main` or `dev` after this change lands
- **THEN** `CHANGELOG.md` and `CHANGELOG_TW.md` both exist at the repository root
- **AND** every entry present in one file has a corresponding entry in the other

#### Scenario: History before v0.3.0 is not duplicated

- **WHEN** a reader opens either changelog file looking for changes released before or in v0.3.0
- **THEN** the file contains no version section for v0.3.0 or earlier
- **AND** the file's header links to the repository's GitHub Releases page as the source for those versions

### Requirement: Changelog structure mirrors the release note

Each changelog file SHALL use `##` for version sections and `###` for package sections, so that promoting every `###` heading to `##` in a version section produces that version's release note body without further editing.

Package section headings SHALL match the names used in release notes: `Core`, `CLI`, `DataList / DataTable`, and backtick-quoted package names such as `` `stats` ``, `` `finance` ``, `` `mkt` ``, `` `datafetch` ``. Entries that apply to the whole release and belong above any package section SHALL be placed as bullets directly under the version heading.

#### Scenario: Unreleased section is the only version section initially

- **WHEN** the changelog files are first created by this change
- **THEN** each file contains exactly one version section, `## Unreleased`
- **AND** that section contains the `CSVReadOptions.RawStrings` entry for pull request #189

#### Scenario: Promoting headings yields a release note

- **WHEN** a maintainer releases a version and copies the `## Unreleased` section body from `CHANGELOG.md`
- **AND** replaces each leading `### ` with `## `
- **THEN** the result matches the heading structure of previously published release notes for this repository

### Requirement: Version sections are renamed at release time

At release time the `## Unreleased` heading SHALL be renamed to the released version number in both files, and a fresh empty `## Unreleased` section SHALL be added above it.

#### Scenario: Releasing a version

- **WHEN** a maintainer publishes version `vX.Y.Z`
- **THEN** the section previously titled `## Unreleased` is titled `## vX.Y.Z` in both changelog files
- **AND** a new empty `## Unreleased` section sits above it in both files

### Requirement: User-visible changes are recorded in the same change

A change that alters behavior a library or CLI user can observe SHALL add its entries to the `## Unreleased` section of both changelog files as part of that same change, not as a follow-up.

Changes with no user-visible effect SHALL NOT add changelog entries. These include internal refactors, test-only edits, dependency bumps with no behavioral effect, formatting, repository assets, and OpenSpec bookkeeping.

#### Scenario: A pull request adds a public API

- **WHEN** a change adds, removes, or alters an exported function, type, method, CLI command, CLI flag, or DSL syntax
- **THEN** the same change adds a bullet describing it under the correct package section of `## Unreleased` in `CHANGELOG.md`
- **AND** adds the matching Traditional Chinese bullet in `CHANGELOG_TW.md`

#### Scenario: A pull request only refactors internals

- **WHEN** a change alters only unexported code, tests, formatting, assets, or OpenSpec artifacts
- **THEN** no changelog entry is added

#### Scenario: A change breaks compatibility

- **WHEN** a change alters existing behavior in a way that requires callers to update their code
- **THEN** its changelog entry is marked as a breaking change in both files, matching how previous release notes marked them

### Requirement: Changelog policy is documented for both agents and humans

The changelog rule SHALL be stated in `AGENTS.md` for AI coding agents and in `CONTRIBUTING.md` for human contributors, and the changelog SHALL be linked from the repository's entry-point documents.

#### Scenario: An agent reads the repository guidance

- **WHEN** an AI coding agent reads `AGENTS.md` before starting work
- **THEN** it finds the rule that a user-visible change updates both changelog files within the same change
- **AND** the rule sits alongside the existing docs-and-skills synchronization policy

#### Scenario: A human contributor reads the contribution guide

- **WHEN** a contributor reads `CONTRIBUTING.md` before opening a pull request
- **THEN** it states when a changelog entry is required, when it is not, and that both language files must be updated together

#### Scenario: A reader looks for the changelog

- **WHEN** a reader opens `README.md`, `README_TW.md`, or `Docs/README.md`
- **THEN** each links to the changelog file matching its own language
- **AND** no copy of the changelog exists under `Docs/`

