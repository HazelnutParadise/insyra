## 1. Implementation
- [x] 1.1 Add CLI delta requirements under `specs/cli-entry/spec.md`
- [x] 1.2 Add command registry delta requirements under `specs/command-registry/spec.md`
- [x] 1.3 Add DSL delta requirements under `specs/dsl-commands/spec.md`
- [x] 1.4 Write `design.md` for user-facing accel control surface and output expectations
- [x] 1.5 Validate the change with `openspec validate add-accel-cli-dsl-surface --strict`
- [x] 1.6 Repair change-local spec text so CLI/DSL requirements are readable and aligned with the implemented control surface
- [x] 1.7 Wire Cobra `--mode` parsing through the shared accel handler and keep `accel plan` output planning-only
- [x] 1.8 Make `accel run <var>` execute a `DataList` or `DataTable` through the execution ledger surface instead of reusing planning-only output
