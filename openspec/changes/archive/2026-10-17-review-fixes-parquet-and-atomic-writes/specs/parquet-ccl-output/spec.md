## MODIFIED Requirements

### Requirement: ApplyCCL writes through a temporary file of its own

`ApplyCCL` SHALL write the new file through a temporary file with a name of its own in the directory of the file that `path` names (the target, when `path` is a symbolic link), and rename it into place only when the whole file was written, so a failure leaves the original as it was, a file named `<path>.tmp` is not touched, and no temporary file is left behind. An input with no rows SHALL leave the original untouched.

#### Scenario: A file named like the old temporary file
- **WHEN** `path + ".tmp"` already holds a file of the user's, and `ApplyCCL` is run
- **THEN** that file's content is unchanged, and the directory holds no files besides these two
