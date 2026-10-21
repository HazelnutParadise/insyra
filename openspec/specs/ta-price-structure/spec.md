# ta-price-structure Specification

## Purpose
Price-structure tools (floor pivot levels, Heikin-Ashi bars, TD Sequential counts, point-and-figure columns) computed from information known on each row.

## Requirements

### Requirement: Pivot levels come from the previous row
`ta.PivotPoints` SHALL return for each row the floor-trader pivot P and the resistances R1 to R3 and supports S1 to S3 computed from the previous row's high, low and close, and NaN on row 0.

#### Scenario: A bar of 12, 8 and 10
- **WHEN** the previous bar has high 12, low 8 and close 10
- **THEN** P is 10, R1 12, R2 14, R3 16, S1 8, S2 6 and S3 4

### Requirement: Heikin-Ashi bars are returned as a table
`ta.HeikinAshi` SHALL return a table with Open, High, Low and Close, the close being the mean of the four prices, the open the midpoint of the previous Heikin-Ashi bar and the first open (open + close) / 2, and the high and low covering the open and close.

#### Scenario: The first bar
- **WHEN** the bar is open 10, high 12, low 9, close 11
- **THEN** the Heikin-Ashi open and close are both 10.5

### Requirement: TD Sequential counts setups and countdowns on closes
`ta.TDSequential` SHALL count 1 to 9 consecutive closes below, or above, the close four rows earlier as a buy, or sell, setup, SHALL start a countdown to 13 after a completed setup that counts rows closing at or beyond the close two rows earlier, and SHALL cancel it when the opposite setup completes.

#### Scenario: A nine-row fall
- **WHEN** nine rows in a row close below the close four rows earlier
- **THEN** the buy setup reads 1 to 9 and the buy countdown starts on the next row

### Requirement: TD Sequential is not DeMark's original rule set
`ta.TDSequential` SHALL compare the close with the close two rows earlier in the countdown, where DeMark's original uses the low or high of two rows earlier, and SHALL start a setup that goes on past 9 again from 1, and its documentation SHALL say so and SHALL NOT claim to match the original.

#### Scenario: A run past nine
- **WHEN** ten rows in a row close below the close four rows earlier
- **THEN** the tenth row's buy setup reads 1

### Requirement: Point-and-figure columns are built from closes
`ta.PointAndFigure` SHALL return the columns of a chart in which a price lies in the box floor(price / boxSize), a column grows with each higher (or lower) box and reverses when the close moves reversal boxes against it, and a new column starts one box beyond the old one's end.

#### Scenario: A rise and a reversal
- **WHEN** closes rise through boxes 10 to 13 and then fall to box 10 with a reversal of 3
- **THEN** a rising column of four boxes is followed by a falling column of three

### Requirement: Heikin-Ashi validates all four series alike
`ta.HeikinAshi` SHALL return an error when any of open, high, low or close is nil, non-numeric, non-finite or not above 0, when the four differ in length, or when a high is below its low.

#### Scenario: A zero low
- **WHEN** the low series holds 0
- **THEN** an error is returned

### Requirement: Unusable input is refused
Each function here SHALL return an error naming the row index for a missing, non-numeric or non-finite value and for a price that must be positive and is not, and an error for a box size that is not above 0, a reversal below 1 and series of different lengths.

#### Scenario: A box size of 0
- **WHEN** boxSize is 0
- **THEN** an error is returned
