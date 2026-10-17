## ADDED Requirements

### Requirement: A duration column round-trips as time.Duration

`Write` SHALL store a column of `time.Duration` as `INT64` nanoseconds, with the Arrow `Duration` type recorded in the file's Arrow schema. `Read`, `ReadFrom`, `ReadColumn`, `Stream` and `StreamFrom` SHALL return those cells as `time.Duration`, and a null cell as `nil`.

#### Scenario: A duration column written and read back
- **WHEN** a column holds `time.Second`, `2 * time.Second` and a null, and is written and then read
- **THEN** the cells read back as `time.Second`, `2 * time.Second` and `nil`

### Requirement: A duration column from another tool keeps its unit

When a file stores its Arrow schema and a column in it is an Arrow duration in seconds, milliseconds, microseconds or nanoseconds, the readers SHALL return that column's cells as `time.Duration` scaled from that unit.

#### Scenario: A column of Arrow durations in seconds
- **WHEN** a file stores its Arrow schema and holds an Arrow `Duration` column in seconds with the values 1, 2 and a null
- **THEN** the cells read back as `time.Second`, `2 * time.Second` and `nil`
