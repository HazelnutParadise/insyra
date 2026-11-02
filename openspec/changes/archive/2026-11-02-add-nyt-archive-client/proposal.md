## Why

Issue #460 has its tone aggregation on both maintained lines, but no monthly NYT article source. Add an authenticated, paced Archive client without changing quant.

## What Changes

- Add NYT(NYTConfig) and NYTClient.Archive/ArchiveContext, following Fugle and YFinance configuration.
- Return monthly article text and publication timestamps in a stable DataTable schema.
- Pace requests, bound 429 retries, reject exhausted local daily budgets immediately, and preserve Context cancellation without exposing credentials.
- Add offline tests, API documentation, and English/Taiwan Traditional Chinese changelogs on dev and 0.4.

## Capabilities

### New Capabilities
- `datafetch-nyt-archive`: authenticated, paced monthly NYT archive retrieval.

### Modified Capabilities
None.

## Impact

Only datafetch, Docs/datafetch.md, the two changelogs, and OpenSpec artifacts. No dependency, existing API, CLI, or quant changes. Both PRs target their maintained line and remain unmerged.
