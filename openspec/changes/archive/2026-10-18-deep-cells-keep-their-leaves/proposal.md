# Proposal: deep-cells-keep-their-leaves

## Why

A value nested more than 32 `[]any` deep reached the encoder's depth limit, where an interface was written as its type alone. Two such values differing only in a deep leaf encoded alike, so `Count` and lookups merged them without a sign (AGENTS.md follow-up 2026-09-14).

## What Changes

- Past the limit the encoder looks through the interface. A scalar is written by value; a container is identified by its address, as the code comment always said.
- A deeper value that is a separate copy is therefore a different value from the original, which is what `==` means for a reference. `TestSharedSubValueIsEncodedPromptly` pinned the old merging at depth 40 and now expects 1.

## Capabilities

### Modified Capabilities

- `cell-identity`: a deep value keeps its leaves.

## Impact

`cell_identity.go`, tests, both CHANGELOGs, the follow-up entry in `AGENTS.md`.
