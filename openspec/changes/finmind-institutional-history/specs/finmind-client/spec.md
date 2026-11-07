## ADDED Requirements

### Requirement: Institutional trader data comes in the exchange's columns
`FinMindClient.InstitutionalFutures` and `InstitutionalOptions` SHALL return one contract's institutional trader rows for the date range in the columns of `TAIFEXClient.InstitutionalFutures` and `InstitutionalOptions`, with the exchange's contract name, trader names, `CALL` or `PUT`, and Net columns equal to Long minus Short.

#### Scenario: The same days from both sources
- **WHEN** TX is read from FinMind for 2025-10-01 and 2025-10-02
- **THEN** every column equals the table built from the exchange's CSV for those days, the net values within one thousand NT dollars

#### Scenario: A foreign investor row
- **WHEN** FinMind names the group 外資
- **THEN** Trader is 外資及陸資

### Requirement: Institutional rows follow the exchange's order
`FinMindClient.InstitutionalFutures` and `InstitutionalOptions` SHALL order rows by date, oldest first, then calls before puts, then 自營商, 投信 and 外資及陸資.

#### Scenario: FinMind's own order
- **WHEN** FinMind sends a day as 投信, 外資, 自營商
- **THEN** the rows of that day are 自營商, 投信, 外資及陸資

### Requirement: The contract name is the one of the kind asked for
`FinMindClient.InstitutionalFutures` SHALL name a contract by its TaiwanFuturesDaily entry and `InstitutionalOptions` by its TaiwanOptionDaily entry, corrected to the exchange's name where FinMind's differs, SHALL read the names again only for a code not in them, and SHALL fail the call without keeping the names when they cannot be read.

#### Scenario: A code of both kinds
- **WHEN** CJ is read as futures
- **THEN** Contract is the futures name 華南金(股票期貨)

#### Scenario: A name FinMind spells differently
- **WHEN** GTF is read and FinMind names it 櫃買期貨
- **THEN** Contract is the exchange's 櫃買指數期貨

#### Scenario: A contract listed after the first call
- **WHEN** a code is not in the names read before
- **THEN** the names are read again and the new code's name is used

#### Scenario: Two calls on one client
- **WHEN** InstitutionalFutures is called twice
- **THEN** the names are requested once

### Requirement: Institutional requests that cannot succeed are refused
`FinMindClient.InstitutionalFutures` and `InstitutionalOptions` SHALL return an error without requesting the data for an empty contract, a `to` before `from`, or a code FinMind has no contract for.

#### Scenario: No contract
- **WHEN** the contract is blank
- **THEN** the call returns an error naming the missing contract code and sends no request

### Requirement: Waiting for the contract names stops with the context
`FinMindClient.InstitutionalFutures` and `InstitutionalOptions` SHALL return the context's error when the context is done while another call on the same client is reading the contract names.

#### Scenario: A deadline while another call reads the names
- **WHEN** another call holds the names and the caller's deadline passes
- **THEN** the call returns context.DeadlineExceeded without waiting for the other call
