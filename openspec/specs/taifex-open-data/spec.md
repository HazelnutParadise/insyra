# taifex-open-data Specification

## Purpose
Reading the Taiwan Futures Exchange's keyless open data, institutional positions, large-trader positions and the put/call ratio, into typed tables for market-sentiment and hedging studies.

## Requirements

### Requirement: The exchange's open data is read into typed tables
`TAIFEXClient` SHALL return, for the latest day, the institutional traders' trading and open interest in futures and in options with the call or put, the large traders' open positions in futures and in options, and the put/call ratio history, each with the date as a time and the counts, values and ratios as numbers.

#### Scenario: A dealer's futures position
- **WHEN** the dealers' TAIFEX futures row is read
- **THEN** TradeNet is -3677, OINet is 3197 and the values are in thousands of NT dollars

#### Scenario: A table answered as CSV
- **WHEN** the large traders' futures table comes back as CSV with Chinese headers and a byte order mark
- **THEN** the same columns are filled as from JSON

### Requirement: What cannot be read is refused
The methods SHALL return an error for a record missing an expected field, a JSON reply that is cut short, an HTTP error status and a client not made by TAIFEX, and SHALL return an empty table with its columns for an empty reply.

#### Scenario: A zero client
- **WHEN** a zero TAIFEXClient is used
- **THEN** an error is returned and no request is sent
