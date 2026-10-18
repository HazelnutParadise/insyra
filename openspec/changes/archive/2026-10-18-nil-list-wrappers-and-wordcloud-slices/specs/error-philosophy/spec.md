## ADDED Requirements

### Requirement: A wrapper holding a nil list is treated as a nil list
The system SHALL treat a value that embeds a nil `*insyra.DataList`, such as `isr.DL`, the way it treats a nil list, in `plot` charts and `stats` functions, and SHALL NOT panic on it.

#### Scenario: A plot chart is given isr.DL
- **WHEN** `plot.CreateBarChart` receives `isr.DL`
- **THEN** it warns and returns nil without panicking

#### Scenario: A stats function is given a wrapper of a nil list
- **WHEN** a hypothesis test receives `isr.UseDL((*insyra.DataList)(nil))`
- **THEN** it treats the list as empty and returns the error an empty list gets

### Requirement: A word cloud counts words by their displayed text
The system SHALL count the cells of a word cloud by the text the chart shows, so that a cell that cannot be a map key, such as a slice, is a word and does not panic.

#### Scenario: A slice cell
- **WHEN** a list holding `[]int{1, 2}` is passed to `plot.CreateWordCloud`
- **THEN** the chart is returned and the cell appears as the word its text gives
