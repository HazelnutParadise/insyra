## ADDED Requirements

### Requirement: Setup prepares the environment ahead of first use

`py.Setup(ctx context.Context) error` SHALL do what the first call of any `py` function does to prepare the environment: download and verify the pinned uv if it is missing, and bring the environment directory to the pinned versions. It SHALL return the setup's error. Once it succeeds, later calls SHALL NOT prepare the environment again, and a `Setup` on a prepared environment SHALL return nil without running uv. `ctx` SHALL bound the download, the sync and the wait for another call's setup; a nil `ctx` SHALL be an error, and a `ctx` that is already done SHALL return `ctx.Err()` before anything starts. Calling `Setup` SHALL stay optional: without it, the first call prepares the environment as before.

#### Scenario: Setup before the first call
- **WHEN** `Setup(ctx)` is called on an environment directory that was never prepared
- **THEN** uv sync runs once and the environment is marked ready, and a later `pyEnvInit` runs no uv command

#### Scenario: Setup twice
- **WHEN** `Setup(ctx)` is called on an environment that is already prepared
- **THEN** it returns nil and runs no uv command

#### Scenario: A failing setup
- **WHEN** the sync fails
- **THEN** `Setup` returns the error, the environment is not marked ready, and the next `Setup` runs the setup again

#### Scenario: Nil or finished context
- **WHEN** `Setup` is called with a nil context, or with a cancelled one on an environment that was never prepared
- **THEN** it returns `errNilContext` or `context.Canceled`, and the environment directory is not touched
