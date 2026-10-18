# Tasks

- [x] Merge a save over the file's current variables using the state read at restore (`cli/env/state.go`, `state_lock.go`).
- [x] Unique temporary file per writer and a short save lock.
- [x] Tests: two managers keep both variables and a removal; twelve concurrent saves keep all; the test fails with the merge disabled.
- [x] `Docs/cli-dsl.md` states it.
- [x] Entries under `## Unreleased` in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] Remove the follow-up from `AGENTS.md`.
- [x] `openspec validate cli-env-saves-merge --strict` passes.
