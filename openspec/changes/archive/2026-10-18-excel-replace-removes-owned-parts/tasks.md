# Tasks

- [x] Remove the replaced sheet's owned parts and their content-type entries (`internal/excelsheet/parts.go`, `replace.go`).
- [x] Test: comment text gone, new table takes the old name, saved file reopens; fails without the change.
- [x] `Docs/DataTable.md` and `Docs/csvxl.md` say the parts are removed.
- [x] Entries under `## Unreleased` in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] Remove the follow-up from `AGENTS.md`.
- [x] `openspec validate excel-replace-removes-owned-parts --strict` passes.
