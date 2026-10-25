## ADDED Requirements

### Requirement: Settings are one optional trailing value

Methods with defaultable settings SHALL take them as one optional trailing options value, refuse more than one before sending a request, and take a required choice such as the ticker type as a parameter.

#### Scenario: Defaults

- **WHEN** `Stock.HistoricalCandles("2330")` is called
- **THEN** it asks for day candles with the API's default range

#### Scenario: Two settings values

- **WHEN** two options values are passed
- **THEN** an error is returned and no request is sent

### Requirement: Documented columns are kept

The tickers, trades and volumes tables SHALL carry every column the documentation lists, as nil cells where the API leaves a field out of every row.

#### Scenario: Tickers without an industry

- **WHEN** every returned ticker lacks an industry
- **THEN** the table still has an industry column of nils
