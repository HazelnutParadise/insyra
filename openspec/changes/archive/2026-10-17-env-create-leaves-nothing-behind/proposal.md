# Proposal: env-create-leaves-nothing-behind

## Why

A review of the `0.4` environment manager found two defects that `dev` shares. When writing an environment's default files failed, `Create` left the half-made directory behind, so the next call saw an environment that did not exist properly. And a default file path that held a directory was skipped as if the file were there. Ported from `review-fixes-lp-stats-cli` on `0.4`.

## What Changes

- `Create` removes the new environment directory when its default files cannot be written, and returns the write error.
- A default file path that holds something other than a regular file is an error naming the file.
- `EnsureDefaultEnvironment` tells a lost creation race from other failures with `errors.Is(err, ErrAlreadyExists)` instead of asking whether the environment exists afterwards.

## Capabilities

### Modified Capabilities

- `env-management`: a failed creation leaves no directory, and a non-regular default file is an error.

## Impact

- `cli/env/manager.go`, `cli/env/create_cleanup_test.go`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
