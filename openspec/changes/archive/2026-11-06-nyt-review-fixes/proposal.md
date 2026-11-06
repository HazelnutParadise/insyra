## Why

The review of the merged NYT Archive client (#671, #672) found that a `Retry-After` had no upper bound (`99999999999` waited 292 years while holding the client), `Retries: 0` still retried once, the daily quota returned the same error as the per-minute limit with no time to retry, future months were sent, one article without `pub_date` failed the month, connection errors lost their cause, error statuses carried no reason, the pacing duplicated `datafetch/internal/limiter`, and the table had no article id. The owner ruled on 2026-11-06 that the daily quota returns a typed error carrying `RetryAt` and waits only when the caller's deadline is after it.

## What Changes

- `NYTQuotaError{RetryAt}`, matching `ErrNYTThrottled`, for the client's exhausted 500-a-day budget and for a 429 whose `Retry-After` is over two minutes. With a Context deadline at or after `RetryAt` the call waits and tries again; otherwise it returns at once.
- A `Retry-After` up to two minutes is waited before a retry; `Retries` counts retries and zero means none.
- Months after the current one are refused without a request.
- New columns `_id`, `news_desk`, `type_of_material`, `word_count` (`int64`), named as in NYT's OpenAPI; a missing `pub_date` is `nil`.
- Error statuses carry the start of the response body with the key removed; connection errors keep their cause without the URL.
- Pacing uses `datafetch/internal/limiter`, which gains an injectable clock; the daily count is unchanged.
- `Docs/datafetch.md` loses its verification notes.

## Impact

- `datafetch/nyt.go`, `datafetch/internal/limiter`, `Docs/datafetch.md`, both changelogs. The client is unreleased, so the column and `Retries` changes need no deprecation.
