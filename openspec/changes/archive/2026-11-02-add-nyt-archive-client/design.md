## Context

Fugle(cfg FugleConfig) and YFinance(cfg YFinanceConfig) create reusable clients. NYT requires an api-key query parameter and returns all articles for one month, potentially about 20 MB. Its official OpenAPI documents 401 and a shared 429 for minute/day exhaustion; no reliable distinction exists in that status alone.

Sources checked on 2026-11-02:
- https://github.com/nytimes/public_api_specs/blob/master/archive_api/archive_api.yaml
- https://github.com/nytimes/public_api_specs/blob/master/archive_api/archive_api.md
- https://developer.nytimes.com/faq (browser access blocked by site-safety policy; current numerical limits unverified).
The legacy official Markdown says 2,000/day and warns limits can change. The issue reporter measured about 5/minute and 500/day on 2026-10-18 with no quota headers. Use those stricter observed limits as client policy, not as a guaranteed official entitlement.

## Goals / Non-Goals

Goals: one request per month on success, offline regression tests, cancellable waits/HTTP, errors.Is auth/quota classification, same addition on both lines.
Non-goals: quant changes, article search, full article bodies, global/environment key configuration, cross-process quota coordination.

## Decisions

- Create `NYT(NYTConfig{APIKey: key})`, then `client.Archive(2026, 10)` or `client.ArchiveContext(ctx, 2026, 10)`. Config has APIKey, BaseURL, Timeout, Interval, Retries. Empty key is rejected; default Interval is 12.1 seconds and smaller intervals are rejected. Retries defaults to one 429 retry (zero config), with no unbounded retry option. No bare NYTArchive convenience function requiring a second configuration route.
- A context-aware channel serializes each client call including its HTTP transaction. A private quota state tracks the next start and up to 500 attempts in a rolling 24-hour window; it is shared by concurrent calls on that client. Daily exhaustion returns ErrNYTThrottled immediately. Only 429 is retried, once by default, after at least 61 seconds without Retry-After. A longer Retry-After is honored with cancellation. Exhausted retries return ErrNYTThrottled; no guessed day-reset sleep.
- HTTP timeout applies per request, not the caller's pacing wait. Redirects are refused. Transport/JSON/server error details are not echoed because they can include the API key. Context errors remain errors.Is-compatible. Never log requests or keys.
- Stable columns: headline (headline.main), abstract, lead_paragraph, section_name, pub_date (time.Time, accepting RFC3339 and colonless numeric UTC offsets), web_url. Missing/null text becomes nil. Empty docs yields zero rows with these columns. Missing/malformed docs and invalid dates are errors, never a false empty success.
- Validate year 1851 through current year and month 1–12 before quota/network use. Do not freeze the stale official example's upper year 2019.
- Local fake HTTP server tests exercise real request construction and conversion. A private clock/wait seam advances time for minute/day tests; real timers verify Context interruption. No public clock or rate-limit bypass options.

## Risks / Trade-offs

- Per-client accounting cannot see another client/process using the same key. Reuse a client; external usage may still trigger 429 and the bounded retry/error handles it.
- A 429 cannot reliably identify minute versus daily exhaustion. Retry after a minute, then return a quota error for caller policy rather than waiting for hours.
- Numerical quotas may change. Document official source mismatch, the date/source of conservative policy, and that this is not an entitlement.
- Response text is provider content, while credentials must never appear in error messages. Decode errors expose only fixed descriptions.

## Migration Plan

Additive only. Port the same files to 0.4, adapting only if necessary for that line's DataTable API. Revert the addition to roll back; no state migration.
