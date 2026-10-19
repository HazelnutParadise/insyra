## ADDED Requirements

### Requirement: A lock being released is waited for on Windows
The system SHALL treat a refusal to create the state lock as a held lock to wait for, and not as a failure, when Windows reports "access denied" for a lock file another process is still deleting.

#### Scenario: Twelve saves at once on Windows
- **WHEN** twelve processes save one environment at the same moment
- **THEN** every save succeeds and every variable is present afterwards
