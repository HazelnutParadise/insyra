# datafetch-nyt-archive Specification

## Purpose
Fetch complete monthly New York Times article metadata through an authenticated client with bounded quota handling, cancellable requests, and a stable DataTable schema.
## Requirements
### Requirement: Monthly archive client
The package SHALL create NYTClient through NYT(NYTConfig) with a required APIKey, and provide Archive(year, month) and ArchiveContext(ctx, year, month). It SHALL reject an uncreated client, invalid configuration, a year before 1851, a month outside 1 through 12, and a month that has not begun anywhere on Earth, without a request.

#### Scenario: Invalid input sends nothing
- **WHEN** the key is empty, the client is uncreated, or the requested year/month is invalid or in the future
- **THEN** an error is returned and no request is sent

### Requirement: Monthly article table
A successful call SHALL GET /{year}/{month}.json with api-key and return _id, headline, abstract, lead_paragraph, section_name, news_desk, type_of_material, word_count, pub_date and web_url columns. Headline SHALL be headline.main, word_count SHALL be int64 and pub_date SHALL retain its timestamp as time.Time. A missing or null field SHALL become nil. Malformed JSON, missing/null docs, a null article and an unreadable timestamp or word count SHALL return errors.

#### Scenario: Articles and empty months
- **WHEN** a valid response has articles or an empty docs array
- **THEN** each article becomes one row in provider order or a zero-row table retaining all ten columns

#### Scenario: Article without a publication date
- **WHEN** one article has no pub_date
- **THEN** its pub_date cell is nil and the rest of the month is returned

### Requirement: Paced bounded quota handling
Each client SHALL space request starts at least 12.1 seconds apart by default, with no configurable faster interval. It SHALL count every HTTP attempt toward a rolling 500-attempt daily budget. A 429 SHALL wait at least 61 seconds, or a longer Retry-After of at most two minutes, before each of Retries retries, zero meaning none, then return ErrNYTThrottled; it SHALL also return ErrNYTThrottled at once when the Context deadline would end the wait.

#### Scenario: Minute limit clears
- **WHEN** Retries is at least one and a server returns 429 then a successful response
- **THEN** the client waits through the minute limit and returns the table

#### Scenario: No retries
- **WHEN** Retries is zero and the server returns 429
- **THEN** one request is sent and the call returns ErrNYTThrottled

#### Scenario: Daily quota cannot cause an hours-long wait
- **WHEN** 500 attempts occurred within 24 hours, or a 429 asks for more than two minutes, and the Context has no deadline
- **THEN** the call returns a *NYTQuotaError matching ErrNYTThrottled without sleeping

### Requirement: Cancellation and credential safety
The client SHALL stop queueing, pacing, retry waits and HTTP work when Context ends, preserving errors.Is compatibility. HTTP 401/403 SHALL match ErrNYTAuth without retries. No log or returned error SHALL contain the API key, and server or transport text SHALL appear only with the key removed; redirects SHALL be refused. Non-loopback endpoints SHALL use HTTPS.

#### Scenario: Cancellation and refused key
- **WHEN** Context ends during a wait/request or the server refuses the key
- **THEN** the call returns the corresponding Context/auth error and sends no further requests

#### Scenario: Hostile error text
- **WHEN** a response or transport failure echoes the credential
- **THEN** the returned error omits it

### Requirement: Daily quota error
An exhausted daily quota SHALL be returned as *NYTQuotaError, which matches ErrNYTThrottled and carries RetryAt, the time a request is expected to be allowed: the oldest counted attempt plus 24 hours, or the time a Retry-After over two minutes names. RetryAt SHALL be no earlier than the client's next request slot. When the Context has a deadline after RetryAt the call SHALL wait until then and try again; otherwise it SHALL return the error at once.

#### Scenario: Deadline long enough
- **WHEN** the budget is exhausted and the Context deadline is after RetryAt
- **THEN** the call waits until RetryAt, sends the request and returns the table

#### Scenario: Deadline too short
- **WHEN** the budget is exhausted and the Context deadline is before RetryAt
- **THEN** the call returns the *NYTQuotaError at once without a request

### Requirement: Error reasons without the key
An error status other than 429 SHALL carry the status code and the start of the response body as one line of printable text with the API key removed. A connection failure SHALL keep its cause for errors.Is and errors.As, and its text SHALL contain neither the URL nor the key.

#### Scenario: Server error with a body
- **WHEN** the server answers 500 with a reason that also echoes the key
- **THEN** the error names the status and the reason, and not the key

