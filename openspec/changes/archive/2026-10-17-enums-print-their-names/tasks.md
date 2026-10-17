## 1. Tests first

- [x] 1.1 Table test over every numeric enum: each constant prints a name, and an out-of-range value prints `TypeName(n)`. Record that it fails on the old code.
- [x] 1.2 Error tests: unknown `VaRMethod`, `OptionType`, `PortfolioObjective`, `LabelSort`, `UnknownPolicy`, `NaNPolicy`, `ResampleFreq` and `Compression` name the value.

## 2. Implementation

- [x] 2.1 Add `String()` to the 19 numeric enums that lack one, and make the existing `String()` methods print `TypeName(n)` for unknown values.
- [x] 2.2 Replace `%d` with `%v` in every error that reports an enum value.
- [x] 2.3 Change the six `uint8` enums to `int`.

## 3. Documentation

- [x] 3.1 AGENTS.md: the rule in Key Conventions, and a follow-up for `stats.CorrelationMethod` and `ml.MetricKind`.
- [x] 3.2 `Docs/` pages that show the changed types.
- [x] 3.3 Add entries to `CHANGELOG.md` and `CHANGELOG_TW.md`. The `uint8` change is marked BREAKING.
- [x] 3.4 Mark QU-1 fixed in `api-review.md`, and add a milestone to `delivery-status.md`.

## 4. Verify

- [x] 4.1 Run gofmt, `go build ./...`, `go vet ./...`, `go test ./...` and `golangci-lint run`.
