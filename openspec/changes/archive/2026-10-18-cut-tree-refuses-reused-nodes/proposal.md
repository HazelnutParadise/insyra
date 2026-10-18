# Proposal: cut-tree-refuses-reused-nodes

## Why

validateTree checks that each merge joins a leaf or an earlier merge but not that each is joined once. Measured on 2026-09-28: merges {-1,-2},{-1,-2} pass and CutTreeByK(tree, 1) returns two clusters with no error.

## What Changes

- CutTreeByK and CutTreeByHeight return an error when a leaf or a merge is joined more than once.

## Capabilities

### Modified Capabilities

- `stats-clustering`: cutting a tree refuses a reused node

## Impact

See `tasks.md`.
