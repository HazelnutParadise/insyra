## ADDED Requirements

### Requirement: Technical indicators take their periods in order

`TechnicalSMA`, `TechnicalRSI`, `TechnicalKDJ`, `TechnicalMACD` and `TechnicalBB` SHALL take the symbol, the timeframe and each period of the indicator as positional parameters in the order of Fugle's query parameters, send them under Fugle's names, and return `date` followed by the indicator's documented columns, which are present even when there is no data.

#### Scenario: KDJ

- **WHEN** TechnicalKDJ is called with FugleTimeframeDay, 9, 3, 3
- **THEN** the request carries timeframe=D, rPeriod=9, kPeriod=3 and dPeriod=3 and the table has the columns date, k, d and j

### Requirement: Technical indicator settings are checked before a request

The technical indicator methods SHALL refuse, without sending a request, a period below 1, a MACD fast period that is not shorter than its slow period, an unknown timeframe, `From` or `To` given alone, `To` before `From`, a date range with a minute timeframe, and more than one options value.

#### Scenario: A range with minute candles

- **WHEN** TechnicalBB is called with FugleTimeframe1Min and a From and To
- **THEN** it returns an error and sends no request
