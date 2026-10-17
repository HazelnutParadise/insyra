## ADDED Requirements

### Requirement: A failed environment creation leaves no directory behind

When writing the default files fails during `Create`, the manager SHALL remove the new environment's directory and SHALL return the error, so that the next `Create` or `NewSession` makes the environment again instead of opening an incomplete one.

#### Scenario: Default files cannot be written
- **WHEN** creating the environment `incomplete` fails while its default files are written
- **THEN** the call returns the error, and the environment's directory does not exist

### Requirement: A default file path that is not a regular file is an error

When a default file path already exists as something other than a regular file, such as a directory, creating an environment SHALL return an error that names the path, and SHALL NOT skip the file.

#### Scenario: A directory where history.txt belongs
- **WHEN** a directory named `history.txt` already exists in the folder being initialised
- **THEN** the call returns an error and the default files are not reported as written

### Requirement: Create reports an existing environment as ErrEnvironmentExists

When the environment already exists, `Create` SHALL return an error for which `errors.Is` matches `ErrEnvironmentExists`, and its message SHALL still name the environment.

#### Scenario: Creating an environment twice
- **WHEN** `Create("known")` is called after an earlier successful `Create("known")` on the same manager
- **THEN** `errors.Is(err, ErrEnvironmentExists)` is true
