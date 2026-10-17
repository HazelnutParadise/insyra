# Proposal: sql-mysql-failed-write-leaves-nothing

## Why

MySQL commits every `CREATE TABLE` and `ALTER TABLE` on its own. `sql-replace-keeps-old-table` fixed Replace for this, and recorded two remaining cases as an `AGENTS.md` follow-up on 2026-09-28: creating a missing table, and adding columns in append mode. Both were measured on 2026-10-17 against MySQL 8.4 in Docker, writing a second batch that passes 65,535 placeholders:

- A `ToSQL` to a missing table returns the error and leaves the new table behind, empty. With `SQLActionIfTableExistsFail`, the next attempt then fails because the table exists.
- An append that adds a column returns the error. The rows roll back, but the new column `extra` stays in the table.

PostgreSQL 16 leaves nothing in either case, because its DDL is transactional.

## What Changes

On a database whose DDL commits on its own (`SQLDialectMySQL`):

- A missing table is written under a staging name, `insyra_new_<16 hex digits>`, inside one transaction. Once every row is in, `RENAME TABLE` puts it under the target's name. A failed write drops the staging table and leaves no table behind.
  - A rename to a name that is taken fails. So a table another session creates meanwhile is kept, and the call returns `already exists`.
  - When the rename returns an error, the call checks whether the staging table is still there, the same way `sql-replace-keeps-old-table` handles a failed swap.
- An append adds its missing columns, then writes the rows in one transaction. If the rows fail, it drops the columns it added. A column that cannot be dropped again is named in the error. The Info line about an added column is logged only once the rows are written.
- Creating a table on MySQL now needs the CREATE, INSERT, ALTER and DROP privileges on the database, as Replace already did. The error says so.
- SQLite and PostgreSQL keep the single transaction.

## Capabilities

### Modified Capabilities

- `sql-table-replace`: what a failed create or append leaves behind.

## Impact

`datatable_to_sql.go`, its tests, the gated server tests, `Docs/DataTable.md`, both CHANGELOGs, and the `AGENTS.md` follow-up, which is removed.
