## ADDED Requirements

### Requirement: A tape can be cleared, closed and given a precision
`Tape.Clear()` SHALL empty the recorded operations and keep parameters and optimizer state. `Tape.Close()` SHALL release the tape's device resources. `Tape.SetPrecision` and `Tape.GetPrecision` SHALL select and report the precision mode; a tape with none set SHALL behave exactly as before, including AdamW's float64 intermediates.

#### Scenario: Clear then train on
- **WHEN** a tape is cleared and a second step is recorded
- **THEN** its Adam step count and moments continue from the first step

#### Scenario: No precision set
- **WHEN** a tape never calls `SetPrecision`
- **THEN** every result equals the result of the previous release
