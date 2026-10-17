## ADDED Requirements

### Requirement: An atomic write follows a symbolic link

On platforms with symbolic links, an atomic write to a path that is a symbolic link SHALL write the file the link resolves to, including a link whose target does not exist yet, and SHALL leave the link in place.

#### Scenario: A link to an existing file
- **WHEN** `path` is a symbolic link to `target`, which holds `old`, and the write stores `new`
- **THEN** `target` holds `new` and `path` is still a symbolic link

#### Scenario: A link to a file that does not exist
- **WHEN** `path` is a symbolic link to `created`, which does not exist, and the write stores `created`
- **THEN** the file `created` holds the new content and `path` is still a symbolic link

### Requirement: An atomic write keeps an existing file's permission bits

On platforms with Unix permission bits, an atomic write to an existing file SHALL keep that file's permission bits, including when the process umask would clear some of them.

#### Scenario: An existing file under a umask
- **WHEN** a file with mode 0664 is written while the umask is 0022
- **THEN** the file is still mode 0664 after the write

### Requirement: An atomic write creates a new file under the umask

On platforms with Unix permission bits, an atomic write that creates a file SHALL give it mode 0666 with the process umask applied.

#### Scenario: A new file under a umask
- **WHEN** a new file is written while the umask is 0027
- **THEN** the file has mode 0640

### Requirement: The temporary file has a short name of its own

An atomic write SHALL create its temporary file in the directory of the resolved target, named `.insyra-<random>.tmp`, so a target whose name is close to the 255-byte limit can still be written.

#### Scenario: A long file name
- **WHEN** an atomic write targets a file whose name is 230 characters long
- **THEN** the write succeeds and the file exists

### Requirement: A panic in the write callback leaves no temporary file

An atomic write whose callback panics SHALL remove its temporary file and SHALL let the panic continue to the caller.

#### Scenario: The callback panics
- **WHEN** the write callback panics with `boom`
- **THEN** the panic reaches the caller and no `.insyra-` temporary file remains in the directory

## MODIFIED Requirements

### Requirement: ToCSV and ToJSON are atomic and honest

`ToCSV` and `ToJSON` SHALL write through a temporary file in the directory of the file the path resolves to, and SHALL rename it into place only after every write and the close succeeded. Any write or flush error SHALL be returned. A failed call SHALL NOT leave a target file or a temporary file behind.

#### Scenario: Write failure
- **WHEN** the underlying writer returns an error
- **THEN** `ToCSV` returns that error

#### Scenario: Missing directory
- **WHEN** the target directory does not exist
- **THEN** an error is returned and no file is added to the directory tree
