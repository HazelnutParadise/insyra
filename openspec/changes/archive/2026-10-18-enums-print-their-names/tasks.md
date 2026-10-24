## 1. Tests first

- [x] 1.1 Table test over every numeric enum: each constant prints a name, and an out-of-range value prints `TypeName(n)`.
- [x] 1.2 Error tests: unknown `VaRMethod`, `OptionType`, `PortfolioObjective`, `LabelSort`, `UnknownPolicy`, `NaNPolicy`, `ResampleFreq` and `Compression` name the value.

## 2. Implementation

- [x] 2.1 Add `String()` to the 19 numeric enums that lack one, and make the existing `String()` methods print `TypeName(n)` for unknown values.
- [x] 2.2 Replace `%d` with `%v` in every error that reports an enum value.

## 3. Documentation

- [x] 3.1 AGENTS.md: the printing rule in Key Conventions.
- [x] 3.2 `Docs/Configuration.md`.
- [x] 3.3 Add entries to `CHANGELOG.md` and `CHANGELOG_TW.md`.

## 4. Verify

- [x] 4.1 Run gofmt, `go build ./...`, `go vet ./...` and `go test ./...`.
