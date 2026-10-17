## ADDED Requirements

### Requirement: A failed create or append leaves the database as it was

On a database whose DDL commits on its own, a `ToSQL` that fails while creating a missing table SHALL leave no table under the target's name and no staging table. A failed append SHALL drop the columns it added, or name the ones it could not drop, and SHALL write none of its rows.

#### Scenario: Create fails on the server
- **WHEN** `ToSQL` to a missing MySQL table fails on its second batch
- **THEN** it returns the error, and neither the table nor a staging table exists

#### Scenario: Append fails on the server
- **WHEN** an append that adds column `extra` fails on its second batch
- **THEN** the table has its old columns and its old rows

#### Scenario: Table created meanwhile
- **WHEN** another session creates the target table just before the rename
- **THEN** the call returns `already exists`, keeps that table, and drops the staging table
