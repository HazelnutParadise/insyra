# Proposal: nested-strings-keep-identity

## Why

A string inside a nested value is written without escaping the characters the identity encoding uses as separators, so Cell([]any{"a,s:b"}) and Cell([]any{"a","b"}) encode alike and Count and Counter treat them as one value.

## What Changes

- Nested strings are written length-prefixed, so no content can stand for a separator.

## Capabilities

### Modified Capabilities

- `cell-identity`: nested strings cannot imitate separators

## Impact

See `tasks.md`.
