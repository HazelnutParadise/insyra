# twstock-shareholding-distribution Specification

## Purpose
Reading the TDCC weekly shareholding distribution, who holds how many shares at each level for every Taiwan security, into a typed table, so the big-holder versus small-holder view needs no hand parsing.

## Requirements

### Requirement: The weekly shareholding distribution is read into a typed table
`TWStockClient.ShareholdingDistribution` SHALL return one row per security and holding level of the TDCC's latest weekly file, with the data date as a time, the code, the level from 1 to 17 with a label, the holders and shares as integers and the percent as a number.

#### Scenario: The largest level of a single-holder security
- **WHEN** a security has one holder holding 422,278,902 shares, all of the custody stock
- **THEN** level 15 has Holders 1, Shares 422278902, Percent 100 and the label "1,000,001 and above"

#### Scenario: The last two levels
- **WHEN** a security's rows are read
- **THEN** level 16 is labelled "difference adjustment" and level 17 "total"

### Requirement: A file that is not the expected one is refused
The method SHALL return an error for a file whose header is not the TDCC's, a row with too few cells, a date that is not YYYYMMDD, a level that is not a number, an empty file, a server error and a client not made by TWStock.

#### Scenario: A wrong header
- **WHEN** the file starts with other column names
- **THEN** an error names the missing header
