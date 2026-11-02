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

Passed: `go build ./...`, `go test ./...`, `go test -race ./datafetch/... -count=1`, `go vet ./...`, `golangci-lint run`, strict validation of the active change before archive, and `openspec validate --specs --strict` after archive. The generated capability Purpose was replaced. The focused datafetch suite was rerun after the boundary fixes. Independent re-review confirmed both fixes without further confirmed findings.
