## ADDED Requirements

### Requirement: A panic in the caller's ReaderAt is an error

`ReadFrom` and `StreamFrom` SHALL return an error, and SHALL NOT end the process, when the `io.ReaderAt` they were given panics during a read.

#### Scenario: A ReaderAt that panics on a page read
- **WHEN** the `io.ReaderAt` passed to `ReadFrom` panics on a read of a valid Parquet file
- **THEN** `ReadFrom` returns an error, `StreamFrom` yields an error, and the process keeps running

### Requirement: Files with large pages are read

`Read`, `Stream`, `Inspect`, `ApplyCCL` and `FilterWithCCL` SHALL read a file whose page is larger than arrow-go v18's default limits of 64 MiB compressed and 256 MiB uncompressed, instead of refusing it.

#### Scenario: A page above the default compressed limit
- **WHEN** a file has a page whose compressed body is larger than 64 MiB
- **THEN** `Read` returns the rows of that page and no error
