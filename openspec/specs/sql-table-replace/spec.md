# sql-table-replace Specification

## Purpose
Says what `ToSQL` with `SQLActionIfTableExistsReplace` leaves behind when the write fails: the old table, untouched. Databases whose DDL is transactional get that from one transaction; MySQL, whose DDL commits on its own, gets it from a staging table renamed into place.
## Requirements
### Requirement: A failed Replace leaves the old table as it was

When `ToSQL` replaces an existing table and the write fails or is cancelled, the old table SHALL keep its name and its rows, and no table created for the write SHALL remain.

#### Scenario: A transactional database

- **WHEN** Replace runs on SQLite with five rows in batches of two and the second `INSERT` fails
- **THEN** the call returns the error, and the table holds its three old rows and is the only table

#### Scenario: The staging path

- **WHEN** the staging-and-rename path runs on SQLite, which the tests use to stand in for MySQL, and the second `INSERT`, the context, the first rename or the second rename fails
- **THEN** the table holds its old rows and no `insyra_new_` or `insyra_old_` table remains

### Requirement: MySQL replaces a table by renaming a staging table into place

On MySQL, Replace of an existing table SHALL refuse the table when another table references it by foreign key, before changing anything. Otherwise it SHALL write the rows into a staging table in the same schema and SHALL put it in place with one `RENAME TABLE` statement, outside any transaction, that also moves the old table to a backup name, then SHALL drop the backup. When the rename returns an error, the call SHALL check whether the staging table is gone and the backup exists, and SHALL finish the replace if so. When the backup cannot be dropped, the call SHALL return an error naming it. When the target did not exist, a table created by another session before the write SHALL be refused, not dropped. SQLite and PostgreSQL SHALL keep dropping and recreating the table inside one transaction.

#### Scenario: A table referenced by another

- **WHEN** another table references the target by foreign key
- **THEN** the call returns an error naming the referencing table, and nothing changes

#### Scenario: A table created meanwhile

- **WHEN** the target did not exist when Replace looked, and another session creates it before the write
- **THEN** the call fails with `already exists` and the other session's table is kept

#### Scenario: The rename statement

- **WHEN** the swap for table `sales` in schema `shop` is built for MySQL
- **THEN** it is ``RENAME TABLE `shop`.`sales` TO `shop`.`insyra_old_x`, `shop`.`insyra_new_x` TO `shop`.`sales` ``

#### Scenario: The old table cannot be dropped

- **WHEN** the rename succeeds and dropping the backup fails
- **THEN** the table holds the new rows, and the error names the backup table, which holds the old rows

