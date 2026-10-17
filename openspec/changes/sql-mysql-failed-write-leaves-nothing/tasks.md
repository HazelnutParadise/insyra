## 1. Tests first
- [x] 1.1 `TestSQLServerFailedCreateLeavesNoTable` and `TestSQLServerFailedAppendLeavesTheTableAsItWas` failed on MySQL 8.4 against the old code on 2026-10-17: the table was left behind, and the columns were `[extra id]`. Both passed on PostgreSQL. The SQLite tests for the same path pass on both versions, because SQLite's DDL is transactional; they cover the new code path, not the defect.

## 2. Implementation
- [x] 2.1 `saveRowsWithCommittedDDL` and `createThroughStagingTable`, sharing `stagingSuffix` and `runSQLCleanup` with Replace.

## 3. Docs and records
- [x] 3.1 `Docs/DataTable.md`, both CHANGELOGs, `delivery-status.md`, and the removed `AGENTS.md` follow-up.

## 4. Verify
- [x] 4.1 gofmt, build, vet, `go test ./...`, `golangci-lint run`, and the server tests against MySQL 8.4 and PostgreSQL 16.
