## ADDED Requirements

### Requirement: A failed IPC server start is the call's error

When the IPC server that carries results back from Python cannot open, the `Run…` call that needed it SHALL return an error that names the address and wraps the listen error, so `errors.Is` sees the underlying cause. The call SHALL return before the Python environment is prepared and before Python starts. A later call SHALL try to open the server again.

#### Scenario: Socket path too long
- **WHEN** `TMPDIR` is 200 characters long on Linux or macOS and `RunCodeContext` is called
- **THEN** it returns an error for which `errors.Is(err, syscall.EINVAL)` is true, and no Python process starts

#### Scenario: The next call tries again
- **WHEN** opening the server failed and the cause is then removed
- **THEN** the next call opens the server and gets its address

### Requirement: The IPC server is open only while a run needs it

Runs in flight at the same time SHALL share one IPC server. When the last of them finishes, the server SHALL close; on Unix this SHALL remove its socket file. An accept loop that stops on an error that is not a timeout SHALL close its listener.

#### Scenario: Concurrent runs share the server
- **WHEN** two runs take the server before either finishes
- **THEN** both get the same address

#### Scenario: The socket file goes with the last run
- **WHEN** on Unix the first of two runs finishes and then the second
- **THEN** the socket file exists after the first finishes and does not exist after the second

#### Scenario: The accept loop fails
- **WHEN** accepting a connection fails with an error that is not a timeout
- **THEN** the accept loop stops and closes the listener, so a Python process connecting afterwards fails at once instead of waiting for an acknowledgement

#### Scenario: A result reaches the store
- **WHEN** a client connects to the server's address and sends a framed message holding an execution ID and a result
- **THEN** the server acknowledges it and the result is stored under that execution ID
