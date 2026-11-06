## 1. Tests

- [x] 1.1 A `Retry-After` of `99999999999` without a deadline returns a `*NYTQuotaError` at once
- [x] 1.2 `Retries: 0` sends one request on 429
- [x] 1.3 An exhausted budget waits only when the deadline is at or after `RetryAt`
- [x] 1.4 A future month sends nothing; a missing `pub_date` is nil; `word_count` reads a number or text
- [x] 1.5 An error status carries the filtered reason; a connection error keeps its cause without the URL

## 2. Implementation

- [x] 2.1 `NYTQuotaError`, the two-minute Retry-After bound, and the deadline-aware quota wait
- [x] 2.2 `Retries` without a default retry; pacing through `limiter.IntervalLimiter`
- [x] 2.3 Month bound, new columns, nil `pub_date`, error reasons

## 3. Documentation

- [x] 3.1 `Docs/datafetch.md` and both changelogs
