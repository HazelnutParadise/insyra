## Why

`Create` makes an environment's folder and then writes `state.json` into it, so a session opening the same new environment from another goroutine or process can read the file while it is still empty or part-written. Since an unreadable `state.json` stops the command, that session fails with `cannot be read: EOF` although nothing is wrong. Found while porting the same stop to `0.4`, where a concurrent-session test hit it on CI.

## What Changes

- `Create` writes each default file to a temporary file and links it into place, so the file appears whole. On a file system without hard links it writes in place, as before.

## Impact

- `cli/env/manager.go` (`writeDefaultFiles`, new `createWhole`). Nothing else changes for an environment that already exists.
