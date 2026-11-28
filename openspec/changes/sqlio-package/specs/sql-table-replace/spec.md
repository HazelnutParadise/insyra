## MODIFIED Requirements

### Requirement: A failed Replace leaves the old table as it was

When `sqlio.Write` with `TableExistsReplace` replaces an existing table and the write fails or is cancelled, the old table SHALL keep its name and its rows, and no table created for the write SHALL remain.

#### Scenario: A transactional database

- **WHEN** Replace runs on SQLite with five rows in batches of two and the second `INSERT` fails
- **THEN** the call returns the error, and the table holds its three old rows and is the only table

#### Scenario: The staging path

- **WHEN** the staging-and-rename path runs on SQLite, which the tests use to stand in for MySQL, and the second `INSERT`, the context, the first rename or the second rename fails
- **THEN** the table holds its old rows and no `insyra_new_` or `insyra_old_` table remains

### Requirement: A failed create or append leaves the database as it was

On a database whose DDL commits on its own, a `sqlio.Write` that fails while creating a missing table SHALL leave no table under the target's name and no staging table. A failed append SHALL drop the columns it added, or name the ones it could not drop, and SHALL write none of its rows.

#### Scenario: Create fails on the server
- **WHEN** `sqlio.Write` to a missing MySQL table fails on its second batch
- **THEN** it returns the error, and neither the table nor a staging table exists

#### Scenario: Append fails on the server
- **WHEN** an append that adds column `extra` fails on its second batch
- **THEN** the table has its old columns and its old rows

#### Scenario: Table created meanwhile
- **WHEN** another session creates the target table just before the rename
- **THEN** the call returns `already exists`, keeps that table, and drops the staging table
