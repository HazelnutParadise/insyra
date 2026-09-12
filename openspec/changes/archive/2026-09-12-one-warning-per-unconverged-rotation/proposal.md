# Proposal: one-warning-per-unconverged-rotation

## Why

`GPForth` and `GPFoblq` log a warning every time a run hits its iteration cap. Under a multi-start search that is one start of many: with `Restarts` now defaulting to 20 (`default-restarts-follow-psych`), a criterion with local minima such as geomin or simplimax can log the warning several times per `FactorAnalysis` call while the solution it returns converged and `RotationConverged` is true. The informed Varimax start `buildStarts` builds warns the same way, though it is a start and not a result. And `insyra.LogWarning` does more than print: it pushes an entry into the global error buffer, so twenty unconverged starts leave twenty entries behind for a result that is fine. Recorded as an `AGENTS.md` follow-up on 2026-09-12.

## What Changes

- `GPForth` and `GPFoblq` report a run that hit the cap at debug level only. Nothing is pushed into the error buffer for it.
- `FaRotations` logs one warning, naming the method, the number of starts and the iteration cap, when — and only when — the solution it chose did not converge. That is the case `RotationConverged` already reports as `false`.
- Two tests pin it: five starts that all fail produce exactly one warning line and one error-buffer entry; twenty starts that converge produce none.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-factor-rotation`: adds the requirement that non-convergence is reported once per search, for the chosen solution only.

## Impact

- A caller who reads the log sees one line per rotation that failed to converge instead of one per failed start; the message names the method and the start count. A caller who reads `RotationConverged` sees no change.
- `GPForth`/`GPFoblq` are internal to `stats`; nothing else calls them.
- Both changelogs gain a `stats` entry; the `AGENTS.md` follow-up is deleted; `delivery-status.md` records the milestone.
