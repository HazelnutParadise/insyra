# Proposal: scaler-json-refuses-empty-reference

## Why

A fitted scaler read from JSON with a column whose `ref` is empty resolves that reference by name and binds to the first unnamed column, shifting its values with no error. Measured on 2026-09-28: a StandardScaler with mean 1 and std 1 turned an unnamed column [5, 7] into [4, 6]. Fit never writes an empty reference.

## What Changes

- Reading scaler JSON fails when any column has an empty `ref`, leaving the receiver as it was.

## Capabilities

### Modified Capabilities

- `scaler-params`: scaler JSON refuses an empty reference

## Impact

See `tasks.md`.
