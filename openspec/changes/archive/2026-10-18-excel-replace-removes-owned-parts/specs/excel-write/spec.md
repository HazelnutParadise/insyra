## ADDED Requirements

### Requirement: A replaced sheet leaves none of its parts in the file
When a sheet is replaced, the system SHALL remove the comments, tables, drawings and the parts only they referenced from the saved file, and SHALL keep a part another sheet still references.

#### Scenario: Old comment text is gone
- **WHEN** a sheet holding a comment is replaced and the workbook is saved
- **THEN** no part of the saved file contains the old comment text

#### Scenario: A new table takes the old table's name
- **WHEN** a sheet holding a table named `Orders` is replaced and a new table named `Orders` is added
- **THEN** adding the table succeeds and the saved file holds one table part
