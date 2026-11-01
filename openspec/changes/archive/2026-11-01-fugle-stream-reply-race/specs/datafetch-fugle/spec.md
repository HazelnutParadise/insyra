## ADDED Requirements

### Requirement: A reply already received answers its command

A stream command SHALL use a reply received before its connection dropped, and SHALL report a dropped connection only when no such reply is queued.

#### Scenario: Reply and drop together

- **WHEN** a command's reply is queued and the connection has dropped
- **THEN** the command applies the reply and returns no error
