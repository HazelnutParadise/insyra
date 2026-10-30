## ADDED Requirements

### Requirement: Historical statistics are one row

`client.Stock.HistoricalStats` SHALL return one row with the documented columns, and an error, not an empty table, when Fugle answers 404 for an emerging stock or a warrant.

#### Scenario: A listed stock

- **WHEN** HistoricalStats is called for 2330
- **THEN** the row has week52High and week52Low and its date is a time

### Requirement: Snapshots keep their time and take typed choices

`SnapshotQuotes`, `SnapshotMovers` and `SnapshotActives` SHALL take the market, and for movers the direction and the change basis, and for actives the trade basis, as positional values refused unless they are among the documented ones, put the snapshot's date, time and market on every row, and keep lastPrice as a column that is nil where the API leaves it out. `SnapshotMovers` SHALL send a non-zero FugleThreshold as its operator query with the value, refuse an unknown operator and a value that is not finite, and send nothing for the zero value.

#### Scenario: A threshold

- **WHEN** SnapshotMovers is given FugleThreshold{Op: FugleGreaterOrEqual, Value: 5}
- **THEN** the request carries gte=5

### Requirement: The market is typed everywhere

`FugleIntradayTickersOptions.Market` SHALL be a FugleMarket and SHALL be refused unless empty or one of TSE, OTC, ESB and TIB.

#### Scenario: A market outside the four

- **WHEN** IntradayTickers is given the market NYSE
- **THEN** an error is returned and no request is sent
