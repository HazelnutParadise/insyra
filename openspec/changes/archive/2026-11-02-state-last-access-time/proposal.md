## Why

Issue #261 has one remaining change: State.LastAccess is a string while EnvironmentInfo already carries time.Time. Changing the field must not prevent existing environments from restoring their variables.

## What Changes

- **BREAKING**: on 0.4, State.LastAccess becomes time.Time and the manager uses it directly.
- Keep the JSON timestamp a string: empty means zero time, valid RFC3339 values retain their instant, and invalid timestamp text means unknown time without rejecting variables.
- Keep existing errors for malformed state documents and non-string, non-null timestamp values.
- Document the migration; dev receives only the notice, keeping its string API.

## Capabilities

### New Capabilities
- `state-last-access`: typed access times with compatible environment persistence.

### Modified Capabilities
None.

## Impact

internal/dsl/env/state.go and manager.go, their public aliases, environment load/save and export/import, Docs/cli-dsl.md and both changelogs. No dependencies or other issue #261 items change.
