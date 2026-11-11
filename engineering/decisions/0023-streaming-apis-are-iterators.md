# 0023 — Streaming APIs are range-over-func iterators

**Status:** accepted
**Date:** 2026-09-24

## Context
`parquet.Stream` returned a pair of channels. A caller that stopped reading early left the producing goroutine blocked, which leaked it.

## Decision
A streaming entry point in Insyra is a range-over-func iterator, not a channel pair. `parquet.Stream` moved on 2026-09-24 with no transition version (`parquet-stream-is-an-iterator`).

## Options rejected
- Keep the channel shape behind a transition version. Its old shape carried the leak.

## Consequences
Any new streaming entry point, the CSV one waiting on #210 included, takes the same shape.
