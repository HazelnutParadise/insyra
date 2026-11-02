# NYT Archive verification

## Scope and contract

The addition is limited to the NYT client, its offline tests, the datafetch documentation, both Unreleased changelogs, and OpenSpec. No quant code or existing API changes. No dependencies added. Sources and quota uncertainty are recorded in design.md and Docs/datafetch.md.

## Test-first evidence

Before implementing NYT, `go test ./datafetch -run TestNYTAcceptance -count=1` failed on the stub with `not implemented` for monthly conversion, empty schema, pacing/429/daily quota, auth, cancellation, invalid inputs and malformed responses. Those tests pass after implementation.

Three later boundary tests were each run failing before their fix:

- `TestNYTAcceptanceLastBudgetAttemptThrottledDoesNotWait`: the 500th attempt returning 429 waited 1m13.1s instead of only the scheduled 12.1s. Now the exhausted local budget returns immediately after that response.
- `TestNYTAcceptanceBodyTimeout`: headers succeeded but the body exceeded the request timeout; the error was `invalid archive JSON` instead of matching `context.DeadlineExceeded`. The decoder now preserves the sentinel without echoing the underlying error.
- `TestNYTAcceptanceOversizedResponse`: valid JSON plus whitespace to 128 MiB followed by garbage was falsely accepted. One extra byte and an explicit limit check now reject it.

## Specification coverage

| Requirement | Evidence |
| --- | --- |
| Monthly archive client | `TestNYTAcceptanceInvalidInputs`, `TestNYTConfigDefaultsAndValidation` |
| Monthly article table | `TestNYTAcceptanceMonth`, `TestNYTAcceptanceEmptyMonth`, `TestNYTAcceptanceColumnOrder`, `TestNYTAcceptanceSchemaErrors`, oversized-response regression |
| Paced bounded quota handling | `TestNYTAcceptancePacingAndQuota`, `TestNYTAcceptanceRefusedAndExhausted`, last-budget regression, `TestNYTConcurrentCallsSharePacing` |
| Cancellation and credential safety | `TestNYTAcceptanceCancelledWait`, `TestNYTCancelsActiveHTTPRequest`, `TestNYTCancelsWhileQueued`, `TestNYTCancelsLongRetryAfter`, `TestNYTHTTPTimeoutAndCredentialSafeErrors`, body-timeout regression |

All HTTP traffic in these tests stays on local fake servers; private clock/wait injection advances quota time without real minute/day sleeps. No real key or NYT request was used. The documented standalone example was extracted and compiled.

## Review

The independent adversarial and documentation review identified the body-timeout and artificial-EOF defects above. Both have failing-before/passing-after regression coverage. Documentation describes a complete usable example, fixed schema, configurable fields and defaults, the scope of quota accounting, and the discrepancy between observed quotas and legacy official documentation.

## Local checks

Passed: `go build ./...`, `go test ./...`, `go test -race ./datafetch/... -count=1`, `go vet ./...`, `golangci-lint run`, strict validation of the active change before archive, and `openspec validate --specs --strict` after archive. The generated capability Purpose was replaced. The focused datafetch suite was rerun after the boundary fixes. On 0.4, go-json v0.10.6 was found to discard underlying reader errors (Stream.read), unlike dev's v0.11.2. The client now records failures at the reader boundary so both versions retain timeout classification. `TestNYTAcceptanceBodyTimeoutAfterJSON` was observed failing with a false nil error on 0.4 before that fix; valid JSON without a completed HTTP body now also fails correctly. Independent re-review confirmed both fixes without further confirmed findings.

## Provider timestamp regression

Remote PR review identified NYT timestamps such as `1980-11-23T05:00:00+0000`, confirmed in the [Gatsby integration author's source](https://github.com/PaulieScanlon/data-champion-ssg-nyc-news/blob/main/src/pages/raw.js), linked by the [Gatsby Archive integration guide](https://www.gatsbyjs.com/blog/fetch-data-from-the-new-york-times-api-using-ssg/). `TestNYTAcceptancePublicationOffsets` failed before the fix for UTC and non-UTC colonless offsets, including fractional seconds. The parser now accepts both RFC3339 and numeric offsets without colons, preserving the instant and offset. Existing RFC3339 and malformed-date tests remain unchanged.

## Endpoint transport regression

A conditional, low-severity remote security finding showed that overriding BaseURL with a non-loopback HTTP endpoint could send the query credential in plaintext. `TestNYTAcceptanceEndpointTransport` was run failing before validation was tightened. HTTPS endpoints and literal IPv4/IPv6 loopback HTTP test servers remain accepted; other plaintext endpoints are rejected before any request. No existing client or dependency was changed.
