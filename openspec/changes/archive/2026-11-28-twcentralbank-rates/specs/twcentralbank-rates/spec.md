## ADDED Requirements

### Requirement: Overnight rates come daily in percent per year
`TWCentralBankClient.OvernightRate` SHALL return the interbank overnight call-loan rate for the days in the range as Date, Rate, High and Low, oldest first, in percent per year as the bank publishes them, with nil for a day without trades and without the file's month summary rows.

#### Scenario: A day without trades
- **WHEN** the file gives "-" for 2001-01-20
- **THEN** that day's Rate, High and Low are nil

#### Scenario: A month summary row
- **WHEN** the file has a row dated 20010100
- **THEN** the table has no row for it

### Requirement: The deposit rate is the five banks' mean or nothing
`TWCentralBankClient.DepositRate` SHALL return, per month, the mean of the one-year fixed-rate time deposit rate of 臺灣銀行, 合作金庫, 第一銀行, 華南銀行 and 彰化銀行 before November 2008 and of 臺灣土地銀行 in place of 彰化銀行 from then, and SHALL return nil for a month the file lacks one of them.

#### Scenario: The month the membership changed
- **WHEN** November 2008 has rows for both 彰化銀行 and 臺灣土地銀行
- **THEN** the mean uses 臺灣土地銀行

#### Scenario: Only one bank in a month
- **WHEN** December 2025 has a row for 臺灣土地銀行 only
- **THEN** its Rate is nil

### Requirement: The central bank's ranges are checked first
`TWCentralBankClient.OvernightRate` and `DepositRate` SHALL treat a zero from or to as an open end and SHALL return an error without a request when to is before from.

#### Scenario: A reversed range
- **WHEN** to is a month before from
- **THEN** the call returns an error and sends no request
