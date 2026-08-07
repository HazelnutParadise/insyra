# Tasks: add-accel-execution-logging

## 1. Mechanism

- [x] 1.1 Add once-per-session info logging at first accelerated execution (device, backend, mode, strategy) and at first qualifying fallback (reason), guarded for concurrent sessions; ineligible-by-caller's-terms outcomes stay debug-only.
- [x] 1.2 Add debug-level per-execution detail: operation, rows, chunk count, per-assignment placement, fallback reason.

- [x] 1.3 `Config.SetAcceleration` logs one info line per actual state transition (enabled/disabled by config), silent on same-value calls — added during acceptance at the owner's request, with a transition-contract test at the root package.

## 2. Verification

- [x] 2.1 Tests capture logger output: N executions → exactly one info line; first qualifying fallback → exactly one info line with the reason; debug level shows per-execution detail; silenced info level writes nothing. Stub probes suffice — no hardware gate needed.
- [x] 2.2 Full `go test ./accel/...` passes with no behavior change beyond logging.

## 3. Docs, changelog, bookkeeping

- [x] 3.1 One line in `Docs/accel.md` about the log lines and how to silence or amplify them; changelog entries in both `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] 3.2 `delivery-status.md` delta; `openspec validate add-accel-execution-logging --strict` passes.
