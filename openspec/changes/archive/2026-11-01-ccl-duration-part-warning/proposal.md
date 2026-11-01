## Why

In 0.4, `DAY`, `HOUR`, `MINUTE` and `SECOND` read a part of a date instead of converting a duration, so the same formula will silently give another answer after an upgrade. The owner ruled on 2026-11-01 that 0.3.6 warns about it, and that means giving people something to move to now.

## What Changes

- `DAY`, `HOUR`, `MINUTE` and `SECOND` log a warning once per process per function. Their results do not change.
- New `DUR(x, unit)`, the same as in 0.4, gives what those functions give for a duration on both lines.
- `DATEPART` is documented as deprecated and removed in 0.4; it logs no warning, because it is the only short way to read those parts on 0.3.x.

## Impact

- `internal/ccl/`, `Docs/CCL.md`, both changelogs, `ccl-type-semantics` spec.
