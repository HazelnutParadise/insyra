# Change: Establish acceleration convergence surface

## Why
The repository has no shared control surface for acceleration work. Without a delivery plan, agent contract, and full phase proposal inventory, the next agent would have to reconstruct scope, sequencing, and defaults from chat history.

## What Changes
- Create `delivery-plan.md` as the shared phase and handoff surface
- Create `AGENTS.md` as the durable operating contract
- Create `CLAUDE.md` as the bootstrap pointer to `AGENTS.md`
- Create the full OpenSpec proposal inventory for the accel phase before any runtime implementation starts

## Impact
- Affected specs: `accel-convergence-surface`
- Affected code: no runtime code; project control surface and planning artifacts only
