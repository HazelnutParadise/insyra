# Proposal: cli-env-lock-windows

## Why

`TestConcurrentSavesNeverLoseAVariable` failed on Windows in CI with `state.lock: Access is denied`, so a save lost its variable. Windows answers "access denied", not "exists", when a file is created while another process's delete of it is pending, and `lockState` treated that as a failure.

## What Changes

- On Windows `lockState` waits on that refusal like on a held lock, up to its usual timeout.

## Impact

`cli/env/state_lock.go`, both CHANGELOGs.
