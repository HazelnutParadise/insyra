# Proposal: sql-replace-keeps-old-table

## Why

`ToSQL` with `IfExists: SQLActionIfTableExistsReplace` ran `DROP TABLE`, `CREATE TABLE` and the batched `INSERT`s inside one `db.Transaction` (#298, SEC-21 in `api-review.md`). That protects the old table only where DDL is transactional. MySQL commits every `CREATE`, `ALTER`, `DROP` and `RENAME` statement on its own and ends the transaction it is in. The MySQL 8.4 Reference Manual lists them under [Statements That Cause an Implicit Commit](https://dev.mysql.com/doc/refman/8.4/en/implicit-commit.html): such a statement implicitly ends any active transaction, "as if you had done a COMMIT". So on MySQL the `DROP` is permanent the moment it runs. An `INSERT` that fails afterwards, for example because `BatchSize × columns` passes the 65,535 placeholder limit, leaves the caller with neither the old table nor the new rows.

This could not be run against MySQL here; no server is installed and none was installed for this change. The reasoning rests on the manual.

## What Changes

- On MySQL, Replace of an existing table writes the rows into a new staging table named `insyra_new_<16 hex digits>` in the same schema, inside a transaction of its own. Only when every row is written does one statement, `RENAME TABLE target TO insyra_old_<same digits>, insyra_new_<…> TO target`, put the staging table in place. The [RENAME TABLE page](https://dev.mysql.com/doc/refman/8.4/en/rename-table.html) states that MySQL performs the rename atomically, so no other session can reach either table while it runs, and that a failed rename changes nothing. The old table is dropped last.
- A failure before the rename, including a cancelled context, drops the staging table and leaves the old table untouched. The cleanup statements run with `context.WithoutCancel`, bounded by 30 seconds, so cancelling the write does not also cancel the cleanup, and a lock held by an interrupted `INSERT` cannot make it wait without end. The single `RENAME TABLE` runs outside a transaction, because a `COMMIT` after it could report a cancellation that happened after the swap. When the rename returns an error, the call checks on a context of its own whether the swap happened, and finishes the replace if it did.
- A table that another table references by foreign key is refused before anything changes. `RENAME TABLE` moves such a reference to the renamed old table, per the manual, so the drop would either fail and leave it there or, with `foreign_key_checks=0`, succeed and leave the reference pointing at nothing. A table that references only itself in the same database is allowed, because the reference moves with the rename. The same name in another database still blocks the replace, since it is a different table.
- Replace on MySQL needs the CREATE, INSERT, ALTER and DROP privileges on the database, because the staging table has a name of its own. An account granted them on the one table only could replace it before and now gets an error before anything changes. On this line the error from a failed `CREATE` is returned as given, and does not list the privileges.
- When the table did not exist at the first look, the create path runs with Fail semantics, so a table another session creates meanwhile is refused instead of dropped, which on MySQL is the step that cannot be undone.
- If the old copy cannot be dropped after the swap, the call returns an error. On this line that error names the target table, not the backup table that still holds the old rows.
- SQLite and PostgreSQL keep one transaction for the drop, create and insert. There DDL is transactional, and dropping and recreating keeps a view that reads the table working. Measured with the SQLite driver the tests use: renaming the table away rewrites a dependent view to the renamed copy, which then breaks when the copy is dropped (`no such table: main.bak`), and renaming a new table in after a drop fails with `error in view v: no such table: main.t`.
- Left on 0.4: the tests for the failure paths and the staging path (this line has one test, a successful Replace on SQLite that leaves no staging table, and the staging path cannot be reached on SQLite here, so it is checked by reading the code only); the error that names the backup table; the privileges in the error for a staging table that cannot be created; the `Docs/DataTable.md` text on the two MySQL cases outside Replace, a table created because it did not exist and columns added in append mode, which are committed before the rows are written; and the AGENTS.md follow-up for those two cases.

## Capabilities

### New Capabilities

- `sql-table-replace`: what Replace does to an existing table when the write fails.

### Modified Capabilities

(none)

## Impact

- `datatable_to_sql.go`: `ToSQLContext`, `replaceThroughStagingTable`, `tableReferencedBy`, and `sqlColumnDefs` and `insertSQLRows`, which both paths share.
- Tests: `datatable_to_sql_replace_test.go` (`TestToSQLReplaceKeepsSQLiteTransactionalBehavior`).
- Docs: `Docs/DataTable.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- Not on this line: `api-review.md` (this repo does not carry it), a `delivery-status.md` entry, and the AGENTS.md follow-up. The agent skills do not describe `ToSQL` and are unchanged.
