## 1. Code and documentation

- [x] 1.1 Delete the deprecated symbols from the code
- [x] 1.2 Update `Docs/` to the surviving names
- [x] 1.3 Add the removals to `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`

## 2. Specs

- [x] 2.1 Remove the requirements that only kept a deprecated name alive
- [x] 2.2 Modify the requirements that also describe a surviving name
- [x] 2.3 Rewrite scenarios that used a removed name as their example
- [x] 2.4 `openspec validate remove-dev-deprecated-apis --strict` passes
- [x] 2.5 Archive the change so the main specs carry the result
