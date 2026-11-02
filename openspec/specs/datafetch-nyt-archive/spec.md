# datafetch-nyt-archive Specification

## Purpose
Fetch complete monthly New York Times article metadata through an authenticated client with bounded quota handling, cancellable requests, and a stable DataTable schema.
## Requirements
### Requirement: Monthly archive client
The package SHALL create NYTClient through NYT(NYTConfig) with a required APIKey, and provide Archive(year, month) and ArchiveContext(ctx, year, month). It SHALL reject an uncreated client, invalid configuration, years outside 1851 through the current year, and months outside 1 through 12 without a request.

#### Scenario: Invalid input sends nothing
- **WHEN** the key is empty, the client is uncreated, or the requested year/month is invalid
- **THEN** an error is returned and no request is sent

### Requirement: Monthly article table
A successful call SHALL GET /{year}/{month}.json with api-key and return headline, abstract, lead_paragraph, section_name, pub_date and web_url columns. Headline SHALL be headline.main and pub_date SHALL retain its timestamp as time.Time. Missing/null text SHALL become nil. Malformed JSON, missing/null docs, and invalid timestamps SHALL return errors.

#### Scenario: Articles and empty months
- **WHEN** a valid response has articles or an empty docs array
- **THEN** each article becomes one row in provider order or a zero-row table retaining all six columns

### Requirement: Paced bounded quota handling
Each client SHALL serialize requests and space starts at least 12.1 seconds apart by default, with no configurable faster interval. It SHALL count every HTTP attempt toward a rolling 500-attempt daily budget and return ErrNYTThrottled immediately once exhausted. A 429 SHALL wait at least 61 seconds (or longer Retry-After) before a bounded retry, then return ErrNYTThrottled on exhaustion.

#### Scenario: Minute limit clears
- **WHEN** a server returns 429 then a successful response
- **THEN** the client waits through the minute limit and returns the table

#### Scenario: Daily quota cannot cause an hours-long wait
- **WHEN** 500 attempts occurred within 24 hours or the configured 429 retries are exhausted
- **THEN** the call returns an error matching ErrNYTThrottled without a day-reset sleep

### Requirement: Cancellation and credential safety
The client SHALL stop queueing, pacing, retry waits and HTTP work when Context ends, preserving errors.Is compatibility. HTTP 401/403 SHALL match ErrNYTAuth without retries. No log or returned error SHALL contain the API key or server/transport text that could echo it; redirects SHALL be refused.

#### Scenario: Cancellation and refused key
- **WHEN** Context ends during a wait/request or the server refuses the key
- **THEN** the call returns the corresponding Context/auth error and sends no further requests

#### Scenario: Hostile error text
- **WHEN** a response or transport failure echoes the credential
- **THEN** the returned error omits it
