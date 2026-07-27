## 1. Changelog files

- [x] 1.1 Create `CHANGELOG.md` with a `# Changelog` title, a header line pointing at GitHub Releases for v0.3.0 and earlier, and a `## Unreleased` section
- [x] 1.2 Seed `CHANGELOG.md` `## Unreleased` with the `CSVReadOptions.RawStrings` entry (#189) under a `### Core` package section
- [x] 1.3 Create `CHANGELOG_TW.md` with the same structure and the Traditional Chinese wording of the same entry
- [x] 1.4 Verify heading levels: version at `##`, package at `###`, so replacing `### ` with `## ` in a version section yields the release note structure

## 2. Contributor-facing policy

- [x] 2.1 Add the changelog rule to `AGENTS.md` inside the "Docs & Skills Must Stay in Sync" section: a user-visible change updates both changelog files in the same change, and it lists what does not warrant an entry
- [x] 2.2 State in `AGENTS.md` that under the OpenSpec workflow the changelog update belongs to the change's own tasks, not a follow-up
- [x] 2.3 Add the same rule to `CONTRIBUTING.md` in wording aimed at human contributors, including the release-time step of renaming `## Unreleased` to the version number and opening a fresh empty one

## 3. Discoverability

- [x] 3.1 Link `CHANGELOG.md` from `README.md`
- [x] 3.2 Link `CHANGELOG_TW.md` from `README_TW.md`
- [x] 3.3 Link the changelog from `Docs/README.md` as an external GitHub link, adding no file under `Docs/`

## 4. Verification

- [x] 4.1 Confirm both changelog files describe the same set of entries
- [x] 4.2 Confirm no `Docs/` mirror file was created and the docsify sidebar generator picks up nothing new. Run `tools/gendocs/gendocs.go` against a **copy** of `Docs/` in a scratch directory, never against `./Docs` — beyond the `--output` sidebar it also overwrites `<dir>/index.html`, `<dir>/_navbar.md`, `<dir>/docs.css`, and `<dir>/logo.webp` in place. Diff the sidebar generated from the pre-change tree against the sidebar generated from the post-change tree.
- [x] 4.3 Re-read the change's own diff and confirm it complies with the rule it introduces (this change is documentation-only, so it adds no changelog entry of its own)

## 5. Manual maintainer step (outside this change's file edits)

- [ ] 5.1 Draft a comment for [Discussion #6](https://github.com/HazelnutParadise/insyra/discussions/6) pointing readers at the changelog, and get explicit approval before posting it
- [ ] 5.2 Post the approved comment
