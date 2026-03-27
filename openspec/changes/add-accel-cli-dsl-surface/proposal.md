# Change: Define accel CLI and DSL surface

## Why
The accel phase requires a user-facing control surface for device inspection, cache inspection, and execution mode selection. This should be planned in the same proposal set as the runtime, not left as a later ad hoc extension.

## What Changes
- Extend the CLI entry surface with an `accel` command group
- Extend the command registry for accel reporting and mode selection handlers
- Extend DSL commands and config hooks for accel mode, device listing, and cache reporting

## Impact
- Affected specs: `cli-entry`, `command-registry`, `dsl-commands`
- Affected code: future Cobra commands, registry handlers, REPL, and DSL execution surface
