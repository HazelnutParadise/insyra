## 1. Contract and failing tests
- [x] 1.1 Check issue #460 and official NYT sources; validate the change strictly.
- [x] 1.2 Add local fake-server tests first and record failure before implementation.

## 2. Implementation and documentation
- [x] 2.1 Implement config/client, monthly table conversion, pacing, bounded quota retries, cancellation, credential-safe errors.
- [x] 2.2 Update Docs/datafetch.md with a runnable example, schema, quotas and source uncertainty; add both Unreleased changelog entries.

## 3. Verification and archive
- [x] 3.1 Run go build ./..., datafetch tests, go vet ./..., golangci-lint run; review code and specification coverage.
- [x] 3.2 Archive this change in the PR, replace generated Purpose, and pass openspec validate --specs --strict.
