## ADDED Requirements

### Requirement: A throttled answer is reported as throttling
The system SHALL return an error that matches `ErrTWStockThrottled` and tells the caller to raise `Interval` and `Retries` when a TWSE or TPEx request is answered with HTTP 307 or 429, and SHALL retry it under the configured `Retries` like any other failed request.

#### Scenario: The exchange answers 307 with its security page
- **WHEN** a request is answered with HTTP 307 and no `Location`
- **THEN** the returned error matches `ErrTWStockThrottled`
- **AND** its message names `Interval` and `Retries`

#### Scenario: A retry gets through
- **WHEN** `Retries` is 2 and the source answers 307, 307, then 200
- **THEN** the call succeeds after three requests

#### Scenario: Other failures are not called throttling
- **WHEN** a request is answered with HTTP 500
- **THEN** the returned error does not match `ErrTWStockThrottled`
