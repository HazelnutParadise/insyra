## ADDED Requirements

### Requirement: A stock's ticker is one row

`FugleStock.IntradayTicker` SHALL request `stock/intraday/ticker/{symbol}`, with `type=oddlot` when its one optional flag is true, refuse more than one flag and an empty or malformed symbol before sending a request, and return one row in which every documented column is present, nil where the API leaves the field out.

#### Scenario: An emerging stock has no price limits

- **WHEN** the API answers without `limitUpPrice` and `limitDownPrice`
- **THEN** the table still has both columns, each holding a nil cell
