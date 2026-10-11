# sql-table-replace Specification

## Purpose
What `ToSQL` with `Replace` does to an existing table on each database: how the new rows replace the old ones, when the old table survives a failure, and what is refused before anything changes.

## Requirements

### Requirement: A failed Replace leaves the old table as it was

When `ToSQL` replaces an existing table and the write fails or is cancelled, the call SHALL leave the old table with its name and rows, and no table created for the write.

#### Scenario: A transactional database
- **WHEN** Replace runs on SQLite with five rows in batches of two and the second `INSERT` fails
- **THEN** the call returns the error, and the table holds its three old rows and is the only table

### Requirement: SQLite and PostgreSQL replace inside one transaction

On SQLite and PostgreSQL, Replace SHALL drop, create and insert inside one transaction.

#### Scenario: A failed insert on PostgreSQL or SQLite
- **WHEN** Replace runs on SQLite or PostgreSQL and an `INSERT` fails
- **THEN** the transaction rolls back, and the table keeps its old rows

### Requirement: MySQL refuses a table that another table references

On MySQL, Replace SHALL refuse an existing table that a foreign key from another table references, before anything changes, unless the only reference is from the table itself in the same database.

#### Scenario: A table referenced by another
- **WHEN** another table in the same database references the target by foreign key
- **THEN** the call returns an error naming the referencing table, and nothing changes

#### Scenario: A table referencing itself
- **WHEN** the target table references itself by foreign key and no other table does
- **THEN** Replace goes ahead

### Requirement: MySQL replaces a table by renaming a staging table into place

On MySQL, Replace SHALL write the rows into a staging table in the same schema and then swap it in with one `RENAME TABLE` statement that also moves the old table to a backup name.

#### Scenario: The rename statement
- **WHEN** the swap for table `sales` in schema `shop` is built for MySQL, with `x` standing for the 16 hex digits of the suffix
- **THEN** the statement is ``RENAME TABLE `shop`.`sales` TO `shop`.`insyra_old_x`, `shop`.`insyra_new_x` TO `shop`.`sales` ``

#### Scenario: A failed insert into the staging table
- **WHEN** an `INSERT` into the staging table fails on MySQL
- **THEN** the staging table is dropped, and the old table keeps its rows

### Requirement: A failed rename is checked before the cleanup

When the `RENAME TABLE` returns an error, the call SHALL check on a context of its own whether the swap happened, and finish the replace if it did.

#### Scenario: The rename did not happen
- **WHEN** the `RENAME TABLE` returns an error and the staging table still exists
- **THEN** the staging table is dropped, the old table keeps its rows, and the error is returned

#### Scenario: The rename happened despite the error
- **WHEN** the `RENAME TABLE` returns an error but the backup exists and the staging table is gone
- **THEN** the call drops the backup, and the table holds the new rows

### Requirement: An old copy that cannot be dropped is reported

When the backup table cannot be dropped after the swap, the call SHALL return an error that names the target table.

#### Scenario: The old table cannot be dropped
- **WHEN** the rename succeeds and dropping the backup fails
- **THEN** the table holds the new rows, and the error names the target table and says the old table remains, without giving the backup's name

### Requirement: A table created meanwhile is refused

When the target did not exist at the first look, Replace SHALL write with Fail semantics, so a table another session creates before the write is refused rather than dropped.

#### Scenario: A table created meanwhile
- **WHEN** the target did not exist when Replace looked, and another session creates it before the write
- **THEN** the call fails with `already exists` and the other session's table is kept
