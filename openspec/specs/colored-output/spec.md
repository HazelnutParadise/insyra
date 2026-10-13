# colored-output Specification

## Purpose
When insyra colours its text output. Log lines and `Show`, `ShowTypes` and `Summary` output are coloured only when they go to a terminal, so files, pipes and buffers receive plain text.

## Requirements

### Requirement: Output is coloured only on a terminal

insyra SHALL add ANSI colour codes to a log line or to `Show`, `ShowTypes` or `Summary` output only when all four conditions hold:
- `Config.GetUseColoredOutput()` is true;
- `NO_COLOR` is unset;
- `TERM` is not `dumb`;
- the destination is a terminal.

Output to anything else SHALL be plain text. That includes a file, a pipe and a `bytes.Buffer`.

#### Scenario: A log line written to a buffer
- **WHEN** coloured output is on and the standard log writes to a buffer
- **AND** `LogError` and `LogFatal` are called
- **THEN** the buffer holds the two lines without any ANSI escape

#### Scenario: Show and Summary written to a buffer
- **WHEN** coloured output is on
- **AND** `DataList.ShowTo`, `DataList.SummaryTo`, `DataTable.ShowTo`, `DataTable.ShowTypesTo` and `DataTable.SummaryTo` each write to a buffer
- **THEN** none of the outputs contains an ANSI escape

#### Scenario: Output on a terminal keeps its colour
- **WHEN** coloured output is on and the destination is a terminal
- **AND** `DataList.ShowTo` writes to it
- **THEN** the output contains ANSI colour codes

#### Scenario: The switches still turn colour off
- **WHEN** the destination is a terminal
- **AND** either `SetUseColoredOutput(false)` is called or `NO_COLOR` is set
- **THEN** the text is not coloured

### Requirement: A coloured log line ends its colour before the newline

A coloured Error or Fatal line SHALL put the colour reset before the line's newline, so the next line starts uncoloured.

#### Scenario: An error line on a terminal
- **WHEN** coloured output is on and the standard log writes to a terminal
- **AND** `LogError("pkg", "Fn", "broken")` is called
- **THEN** the output is `\x1b[31m[insyra - Error] pkg.Fn: Broken\x1b[0m` followed by a newline
