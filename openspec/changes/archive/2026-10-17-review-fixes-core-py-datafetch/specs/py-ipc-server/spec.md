## ADDED Requirements

### Requirement: A stopped IPC server is replaced by the next run

When the accept loop of the IPC server stops on an error that is not a timeout, the server SHALL forget that listener and its address, so the next `Run…` call that needs the server SHALL open a new one and MUST NOT be given the address of the stopped one.

#### Scenario: A run after the accept loop stopped

- **WHEN** the accept loop stops on a non-timeout error while runs are using the server, and a later run needs the server
- **THEN** the later run gets a new server address, and the stopped server's address is not used
