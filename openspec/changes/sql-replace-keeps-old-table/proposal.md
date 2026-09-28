# Proposal: sql-replace-keeps-old-table

## Why

`ToSQL` with `IfExists: SQLActionIfTableExistsReplace` ran `DROP TABLE`, `CREATE TABLE` and the batched `INSERT`s inside one `db.Transaction` (#298, SEC-21 in `api-review.md`). That protects the old table only where DDL is transactional. MySQL commits every `CREATE`, `ALTER`, `DROP` and `RENAME` statement on its own and ends the transaction it is in. The MySQL 8.4 Reference Manual lists them under [Statements That Cause an Implicit Commit](https://dev.mysql.com/doc/refman/8.4/en/implicit-commit.html): such a statement implicitly ends any active transaction, "as if you had done a COMMIT". So on MySQL the `DROP` is permanent the moment it runs. An `INSERT` that fails afterwards, for example because `BatchSize × columns` passes the 65,535 placeholder limit, leaves the caller with neither the old table nor the new rows.

This could not be run against MySQL here; no server is installed and none was installed for this change. The reasoning rests on the manual.

## What Changes

- On MySQL, Replace of an existing table writes the rows into a new staging table named `insyra_new_<16 hex digits>` in the same schema, inside a transaction of its own. Only when every row is written does one statement, `RENAME TABLE target TO insyra_old_<same digits>, insyra_new_<…> TO target`, put the staging table in place. The [RENAME TABLE page](https://dev.mysql.com/doc/refman/8.4/en/rename-table.html) states that MySQL performs the rename atomically, so no other session can reach either table while it runs, and that a failed rename changes nothing. The old table is dropped last.
- A failure before the rename, including a cancelled context, drops the staging table and leaves the old table untouched. Each cleanup statement runs with `context.WithoutCancel` bounded by 30 seconds, so cancelling the write does not also cancel the cleanup, and a lock held by an interrupted `INSERT` cannot make it wait without end. The single `RENAME TABLE` runs outside a transaction: it commits on its own, and a `COMMIT` after it could report a cancellation that happened after the swap. When the rename returns an error, the call looks at which tables exist to tell whether it took effect, and finishes the replace if it did. If the old copy cannot be dropped after the swap, the call returns an error naming it.
- A table that another table references by foreign key is refused before anything changes. `RENAME TABLE` moves such a reference to the renamed old table, per the manual, so the drop would either fail and leave it there or, with `foreign_key_checks=0`, succeed and leave the reference pointing at nothing. The old code refused the same case at the `DROP`. A reference from the table to itself moves and goes with it, and is allowed.
- The staging table has a name of its own, so Replace on MySQL now needs the CREATE, INSERT, ALTER and DROP privileges on the database. An account granted them on the one table only could replace it before and now gets an error before anything changes; the error says which privileges are needed.
- When the table did not exist at the first look, the create path runs with Fail semantics, so a table another session creates meanwhile is refused instead of dropped, which on MySQL is the step that cannot be undone.
- SQLite and PostgreSQL keep the single transaction. There DDL is transactional, and dropping and recreating keeps a view that reads the table working. Measured with the SQLite driver the tests use: renaming the table away rewrites a dependent view to the renamed copy, which then breaks when the copy is dropped (`no such table: main.bak`), and renaming a new table in after a drop fails with `error in view v: no such table: main.t`. On those two, the rename path runs only in tests.
- `Docs/DataTable.md` no longer claims that every statement runs in one rollback-able transaction on every database. It describes what MySQL leaves behind, including the two MySQL cases this change does not cover: a table created because it did not exist, and columns added in append mode, are committed before the rows are written. Both are recorded as an `AGENTS.md` follow-up.

## Capabilities

### New Capabilities

- `sql-table-replace`: what Replace does to an existing table when the write fails.

### Modified Capabilities

(none)

## Impact

- `datatable_to_sql.go`: `toSQL`, `replaceThroughStagingTable`, `renameSwapStatements`, and `resolveSQLColumnTypes`, `sqlColumnDefs`, `insertSQLRows` taken out of `saveRowsToDB` so both paths share them. New `datatable_to_sql_replace_test.go`.
- `Docs/DataTable.md`, both changelogs, `api-review.md`, `AGENTS.md`, `delivery-status.md`.
- The agent skills do not describe `ToSQL` and are unchanged.
