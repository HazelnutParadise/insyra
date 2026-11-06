## Why

`Create` writes an environment's `state.json` after it makes the folder, so a session opening the same new environment from another goroutine or process can read the file while it is still empty. Since an unreadable `state.json` stops the command, that read fails with `EOF` although the environment holds nothing. Found while porting the same stop to `0.4`, where a concurrent-session test hit it.

## What Changes

- `Manager.OpenVariables` starts an environment whose `state.json` is empty with no variables, the same as one with no file. An empty file holds no variables, so nothing can be lost.

## Impact

- `cli/env/state_open.go`. A non-empty file that cannot be read still stops the command.
