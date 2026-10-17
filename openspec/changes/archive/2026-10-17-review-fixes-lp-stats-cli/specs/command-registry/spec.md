## ADDED Requirements

### Requirement: Register refuses a negative argument limit

`Register` SHALL refuse a handler whose argument limit has a negative maximum, a negative form position, or a negative count for any form, returning an error and leaving the command unregistered.

#### Scenario: A negative maximum
- **WHEN** a handler declares `MaxArgs(-1)` and is registered
- **THEN** `Register` returns an error and the command is not registered

#### Scenario: A negative count for one form
- **WHEN** a handler declares `FormArgs(map[string]int{"form": -1})` and is registered
- **THEN** `Register` returns an error and the command is not registered
