## ADDED Requirements

### Requirement: WriteTo reports a row group it cannot start or write

`WriteTo` and `WriteToContext` SHALL return the error of a row group that cannot be started or written, and SHALL NOT panic.

#### Scenario: A destination that refuses a row group
- **WHEN** the destination refuses a write after the header, and the table is written with `RowGroupSize: 1`
- **THEN** the call returns that error and does not panic

### Requirement: A cancellation during the footer is reported

`WriteToContext` SHALL return `ctx.Err()` when `ctx` is done while the footer is written, after every row group was written.

#### Scenario: Cancelled as the footer is accepted
- **WHEN** `ctx` is cancelled as the last bytes of the footer are accepted by the destination
- **THEN** `WriteToContext` returns `context.Canceled`, and the destination holds every byte of the file

## MODIFIED Requirements

### Requirement: Writes to one path do not mix

`Write` and `WriteContext` SHALL write through a temporary file with a name of its own in the directory of the file that `path` names (the target, when `path` is a symbolic link), so two writes to the same path at once each leave either their own complete file or the other's at `path`, never a mix, a write the operating system refuses to rename leaves no temporary file, and a file the caller keeps at `<path>.tmp` is not touched.

#### Scenario: Two writers
- **WHEN** two goroutines each write to the same path 20 times in a row, one writing 1,000 rows and one writing 3,000 rows
- **THEN** each side has at least one write that returns nil (Windows may refuse one of two renames onto the same target at once), the final file is readable with 1,000 or 3,000 rows, and no other file is left in the directory

#### Scenario: A file named like the old temporary file
- **WHEN** `path + ".tmp"` already holds a file of the user's, and `Write(dt, path)` is called
- **THEN** that file's content is unchanged
