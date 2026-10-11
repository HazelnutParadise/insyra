## ADDED Requirements

### Requirement: Commands declare their one-shot flags at registration

`BuildCobraCommands` SHALL register and forward each one-shot flag a `CommandHandler` declares in `Flags`, without naming any command.

#### Scenario: env clear keeps its history flag
- **WHEN** the shell runs `insyra env clear a --keep-history`
- **THEN** `env`'s Run receives `clear a --keep-history`

#### Scenario: A flag outside its form is dropped
- **WHEN** the shell runs `insyra env clear a --force`
- **THEN** `env`'s Run receives `clear a`

#### Scenario: A flag with a value
- **WHEN** the shell runs `insyra accel devices --mode cpu`
- **THEN** `accel`'s Run receives `devices --mode cpu`

### Requirement: Register refuses a flag it cannot register

`Register` SHALL refuse a handler with a flag that has no name, or whose name is declared twice, without registering the command.

#### Scenario: A flag declared twice
- **WHEN** a handler declares `--force` twice and is registered
- **THEN** `Register` returns an error and the command is not registered

#### Scenario: A flag with no name
- **WHEN** a handler declares a flag with an empty name and is registered
- **THEN** `Register` returns an error and the command is not registered

### Requirement: accel reads its mode from the session's manager

When `accel` runs without `--mode`, it SHALL read the default mode from the session's environment manager (`ExecContext.Env`), falling back to `Default()` only when the session has none.

#### Scenario: A session with its own manager
- **WHEN** `Default()`'s config sets `accel-mode` to `gpu`, the session's manager sets it to `cpu`, and `accel` runs without `--mode`
- **THEN** the mode is `cpu`

#### Scenario: A session without a manager
- **WHEN** `accel` runs without `--mode` in a session that has no manager, and `Default()`'s config sets `accel-mode` to `gpu`
- **THEN** the mode is `gpu`
