## ADDED Requirements

### Requirement: Environment operations have one public name

Every environment operation SHALL be a method of `Manager`. The package-level functions that only call the same method on `Default()` SHALL carry `Deprecated: use Default().<Name> instead.` and SHALL keep acting on `Default()` until they are removed. No caller in the repository SHALL use them, apart from the test that pins them.

#### Scenario: A deprecated wrapper keeps its meaning
- **WHEN** a caller runs `SetBasePath(dir)` and then `Create("wrapped")`
- **THEN** `Default().Exists("wrapped")` is true and `Default().BasePath()` is `dir`
