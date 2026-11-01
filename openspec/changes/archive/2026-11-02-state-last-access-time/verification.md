# Verification

All three requirements and their scenarios are covered by `internal/dsl/env/state_last_access_test.go`. The tests exercise LoadState, RestoreVariables, Info, Export, Import and SaveState. Existing typed-state tests cover the variable codecs and import protection.

## Test-first evidence

Before implementation, the new compatibility tests failed because LastAccess was a string rather than time.Time, including the actual old-writer fixture. After implementation, all environment tests passed. Additional tests first exposed unrepresentable year/time-zone output and a parsed +24:00 offset that JSON cannot write; the writer now returns the standard time error for unrepresentable Go values and the reader treats unrepresentable timestamp text as unknown time.

## Checks

- `go build ./...`: passed.
- `go test ./internal/dsl/... ./cli/... ./engine/dsl/...`: passed.
- `go test ./internal/dsl/env -count=1`: passed, independently rerun during review.
- `go vet ./...`: passed.
- `golangci-lint run`: 0 issues.
- `openspec validate state-last-access-time --strict`: passed before archive.
- `openspec validate --specs --strict`: 235 passed, 0 failed after archive.
- The unmodified 0.3.6 writer produced the committed fixture. The actual dev reader also restored a file produced by the 0.4 writer, including both the large integer and text variable.

## Review and boundaries

Independent compatibility review identified the +24:00 parser/marshaler mismatch; the regression test and fix were reviewed again with no remaining findings. The change leaves malformed-document and non-string timestamp failures on the existing paths, and does not address other parts of #261. Timestamp strings are semantically compatible, not byte-for-byte preserved: invalid text becomes unknown time and valid formatting may be normalized.

Official behavior checked: [time.Time JSON](https://pkg.go.dev/time#Time.UnmarshalJSON), [time.Time.MarshalJSON](https://pkg.go.dev/time#Time.MarshalJSON), and [Decoder.UseNumber](https://pkg.go.dev/encoding/json#Decoder.UseNumber). The +24:00 distinction was also reproduced against the installed Go toolchain.
