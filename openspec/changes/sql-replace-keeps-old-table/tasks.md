# Tasks: sql-replace-keeps-old-table

## 1. Tests first

- [x] 1.1 `datatable_to_sql_replace_test.go`, on a file-backed SQLite database so every pooled connection sees the same tables: through `ToSQL`, a Replace whose second `INSERT` fails keeps the old rows. This passed on the old code as well, because SQLite's DDL is transactional; it pins that path.
- [x] 1.2 Same file, through `toSQL` with the staging path forced on SQLite: a failed second `INSERT`, a context cancelled after the first `INSERT`, a failed first rename and a failed second rename each keep the old rows and leave no other table; a successful replace swaps the new rows in and leaves no other table; a missing table is created; a failed drop of the backup is reported with its name while the new rows are in place. Against the old code the file does not compile: `toSQL` is undefined. With the cleanup on the caller's context instead of `context.WithoutCancel`, the cancel test failed with the staging table left behind; with the swap outside a transaction, the second-rename test failed.
- [x] 1.3 Same file: `renameSwapStatements` builds MySQL's one `RENAME TABLE` statement, with and without a schema, and the two `ALTER TABLE` statements elsewhere.
- [x] 1.4 After an adversarial review (Claude Opus through the Agent tool; agy's quota was exhausted): a table referenced by another is refused with nothing changed; a self-referencing table is replaced; a staging table that cannot be created stops the call with the privileges it needs and nothing to drop; a table that appears between the two existence checks is refused, not dropped; `renameSwapHappened` tells a completed swap from one that did not happen. The referenced-table and appearing-table tests failed with their fix removed. The review also found `TestToSQLReplaceByRenameCleansUpAfterCancel` failing 22 times in 300 runs with `SQLITE_BUSY`, because the cleanup met the rollback `database/sql` runs in the background after a cancellation; with a busy timeout and WAL on the test database, 300 runs of every staged-path test pass.

## 2. Implementation

- [x] 2.1 `datatable_to_sql.go`: `ToSQLContext` goes through `toSQL`, which sets the default batch size and, for Replace of an existing table on MySQL, calls `replaceThroughStagingTable`, and otherwise creates the table with Fail semantics; the column typing, `CREATE TABLE` column list and batched `INSERT` are shared helpers; `tableReferencedBy` refuses a referenced table; MySQL's single rename runs outside a transaction and the two-statement swap inside one; a rename error is checked with `renameSwapHappened`; each cleanup statement runs with `context.WithoutCancel` and a 30-second bound.
- [x] 2.2 Doc comments on the three `SQLActionIfTableExists` values.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`: the ToSQL Behavior, the Replace value and the Returns line say what a failure leaves behind on each database, with the MySQL manual cited.
- [x] 3.2 Skills: no change.
- [x] 3.3 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core.
- [x] 3.4 `api-review.md`: SEC-21 fixed.
- [x] 3.5 `AGENTS.md`: a follow-up for the MySQL cases outside Replace. `delivery-status.md`: a Latest Milestones entry.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` (0 issues), `openspec validate sql-replace-keeps-old-table --strict`. MySQL itself was not run: no server was available and none was installed.
