# 0017 — Every non-test file uses one JSON library

**Status:** accepted
**Date:** 2026-09-28

## Context
Two JSON libraries disagreed. Before `goccy/go-json` v0.11 the same small number was written `1e-07` on one path and `1e-7` on another.

## Decision
Every non-test file reads and writes JSON through one library, currently `github.com/goccy/go-json`, and `depguard` fails on `encoding/json` or another JSON library outside tests. A library measured faster on `BenchmarkJSONPaths` that passes every `json-codec-conformance` test replaces it everywhere in one change. The owner decided on 2026-09-28 (#406).

## Options rejected
- Keep `encoding/json` on some paths. The two disagreed on output, and go-json is faster on insyra's paths (`ToJSON_Bytes` 66 to 45 ms, `ReadJSON` 129 to 90 ms on a 14 MB table, v0.11.1 against v0.10.6).

## Consequences
The conformance spec and the committed benchmark make both halves of the rule checkable. A go-json update that changes what insyra writes or accepts lands as its own change, under the dependency rule in `AGENTS.md`.
