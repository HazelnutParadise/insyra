## ADDED Requirements

### Requirement: Environment operations have one public name

Every environment operation SHALL be a method of `Manager`; the package-level wrappers that only call `Default()` carry a `Deprecated:` notice and keep acting on `Default()` until 0.4 removes them.

#### Scenario: A deprecated wrapper keeps its meaning
- **WHEN** a caller runs `SetBasePath(dir)` and then `Create("wrapped")`
- **THEN** `Default().Exists("wrapped")` is true and `Default().BasePath()` is `dir`
