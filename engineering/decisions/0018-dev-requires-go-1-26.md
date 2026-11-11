# 0018 — `dev` requires Go 1.26

**Status:** accepted
**Date:** 2026-10-10

## Context
Go 1.25 left support when Go 1.27 was released. On 2026-10-08 eleven standard-library advisories reachable from insyra code were published with fixes only in go1.26.9 and go1.27.2, and the `x/net` fix for five of them declares `go 1.26.0`. The Vulnerability Scan could not turn green on 1.25, so no release could ship.

## Decision
The `go` directive on `dev` moved from 1.25.12 to 1.26.9 (`require-go-1-26`), and the modules the 1.25 directive held back moved in their own changes after it. The owner decided on 2026-10-10, recorded on #203.

## Options rejected
- Stay on 1.25 and wait for backported fixes. None were published, and the `x/net` fix cannot be taken on 1.25.

## Consequences
The minimum Go version is part of the promise to downstream users; raising it again is an explicit decision, not a side effect of a dependency refresh (`AGENTS.md`, Dependency Refresh Before a Release).
