# Proposal: engineering-docs-layout

## Why

The project's working documents for its developers are spread over two files at the repository root. `ENG.md` holds the architecture together with the reasoning behind every standing choice, and `delivery-status.md` has grown to over 350 lines mixing the current phase, about 200 dated milestone entries and about 60 decisions, most of them about work that has since merged and many already recorded in the specs and archived changes they produced. An agent arriving at the project is told to read the whole status file first, and has to sort out for itself which of it still holds.

Each kind of content now gets one place, under `engineering/`:

- How the system is built: `engineering/architecture.md`, describing the present.
- Why a hard-to-reverse choice was made: one record per choice under `engineering/decisions/`, never rewritten after it is accepted.
- What is left to do and what it waits for: the open OpenSpec changes, as before, with what each waits for written in its `proposal.md`.
- Unfinished work that no change covers: `engineering/handoff.md`, present only while there is some.

The history of finished milestones is not carried over. Git, `openspec/changes/archive/` and the changelogs already hold it.

## What Changes

- `ENG.md` moves to `engineering/architecture.md`, keeps the system as it is now, and links to the decision records instead of repeating their reasoning.
- Every decision in `delivery-status.md` that still holds and is not already recorded in a spec or an archived change becomes a decision record under `engineering/decisions/`. Working rules move to `AGENTS.md`, facts about one machine go to the maintainers' own notes, and unfinished work with no change goes to `engineering/handoff.md`.
- `delivery-status.md` is deleted.
- `engineering/README.md` indexes the directory.
- `AGENTS.md`'s required artifacts and the acceleration operating contract name the new locations: a measurement is recorded in the decision record or the change it decides, and a handoff in the change it concerns.
- The four main specs that name `ENG.md` or `delivery-status.md` name the new locations.

## Capabilities

### Modified Capabilities

- `accel-convergence-surface`: the control surface for acceleration work is `AGENTS.md`, `engineering/` and the open OpenSpec changes, and a handoff is written in the change it concerns.
- `accel-scheduler`: the saturation measurement and its decision are written in a decision record.
- `nn-edge-sum`: the device measurement and its verdict are written in a decision record.
- `nn-training-frontdoor`: the Fit parity requirement names `engineering/architecture.md` as the home of the gate it extends.

## Impact

`ENG.md`, `delivery-status.md`, `engineering/`, `AGENTS.md`, `.github/workflows/nn-data-gates.yml` (a comment), two code comments that cite the old files, the review checklist in `project/api-review.md`, the four specs above. No Go behaviour changes and nothing a user of the library or the CLI sees, so no changelog entry.

Blocked by: none.
