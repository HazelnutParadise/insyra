## 1. Tests first
- [x] 1.1 `TestToSQLWritesAnUnnamedColumnUnderItsLetter`, `TestToSQLRefusesAnUnnamedColumnWhoseLetterIsTaken` and `TestSQLServerWritesAnUnnamedColumn`. Both SQLite tests failed against the old code on 2026-10-17. The server failure is the one recorded in the follow-up.

## 2. Implementation
- [x] 2.1 `nameUnnamedSQLColumns` in `toSQL`.

## 3. Docs and records
- [x] 3.1 `Docs/DataTable.md`, both CHANGELOGs, `delivery-status.md`, and the removed `AGENTS.md` follow-up.

## 4. Verify
- [x] 4.1 gofmt, build, vet, `go test ./...`, `golangci-lint run`, and the server tests against MySQL 8.4 and PostgreSQL 16.
