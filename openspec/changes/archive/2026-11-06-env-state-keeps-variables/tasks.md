## 1. Tests

- [x] 1.1 A `lastAccess` of another JSON type loads as zero time and keeps the variables through a save
- [x] 1.2 A struct embedding `State` keeps its own fields through JSON
- [x] 1.3 `OpenVariables` refuses an unreadable file without changing it, and starts a missing one empty; two sessions opening a missing environment at once both succeed
- [x] 1.4 `NewSession` on an unreadable file fails naming it and leaves it unchanged

## 2. Implementation

- [x] 2.1 Private stored-layout type shared by reading, `SaveVariables`, `Export` and `Import`; `State` loses its JSON methods
- [x] 2.2 `OpenVariables`, used by the one-shot command, the REPL, `NewSession`, `env open` and `env import`

## 3. Documentation

- [x] 3.1 `Docs/cli-dsl.md`, both changelogs, `project/api-review.md` CL-3, `delivery-status.md`, and the `AGENTS.md` follow-up removed
