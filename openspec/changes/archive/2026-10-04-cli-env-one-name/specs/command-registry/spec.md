## ADDED Requirements

### Requirement: Commands declare their one-shot flags at registration

A `CommandHandler` SHALL declare the flags its one-shot form takes in `Flags`. `BuildCobraCommands` SHALL register each declared flag with Cobra under its name and help text, and SHALL append each one that is set to the arguments `Run` receives, in declared order: a switch as `--name`, a flag that takes a value as `--name value` when the value is not blank. A flag with a `Form` SHALL be appended only when the first argument is that word, in any letter case. `BuildCobraCommands` SHALL NOT single out a command by name to add or forward a flag.

`Register` SHALL refuse a handler with a flag that has no name or a name declared twice, with an error and without registering it.

#### Scenario: A flag declared twice
- **WHEN** a handler declares `--force` twice and is registered
- **THEN** `Register` returns an error and the command is not registered

#### Scenario: env clear keeps its history flag
- **WHEN** the shell runs `insyra env clear a --keep-history`
- **THEN** `env`'s Run receives `clear a --keep-history`

#### Scenario: A flag outside its form is dropped
- **WHEN** the shell runs `insyra env clear a --force`
- **THEN** `env`'s Run receives `clear a`

#### Scenario: A flag with a value
- **WHEN** the shell runs `insyra accel devices --mode cpu`
- **THEN** `accel`'s Run receives `devices --mode cpu`

### Requirement: accel reads its mode from the session's manager

When `accel` is run without `--mode`, it SHALL read the default mode from the global config of the session's environment manager (`ExecContext.Env`), and from `Default()` only when the session has none.

#### Scenario: A session with its own manager
- **WHEN** `Default()`'s config sets `accel-mode` to `gpu`, the session's manager sets it to `cpu`, and `accel` runs without `--mode`
- **THEN** the mode is `cpu`
