## 1. Contract and failing tests

- [x] 1.1 Add public API skeleton and offline generated writer tests; record failures before implementation.

## 2. Reader

- [x] 2.1 Implement metadata-only path and ReaderAt inspection, including dictionaries without fetching values.
- [x] 2.2 Implement selected, bounded streaming, cancellation, ownership and explicit unsupported-type errors matching Parquet cells.
- [x] 2.3 Bound output cell slots and expanded string/binary payloads before allocation; cover dictionary expansion and null-only batches with failing tests first.

- [x] 2.4 Match the 0.4 column-selector contract with index strings, insyra.Name and int positions; prove B differs from Name("B") with tests first.

- [x] 2.5 Reject overlapping or duplicate dictionary/record block extents before body dispatch, with failing generated-file regressions first.

- [x] 2.6 Bound copied schema/name/key-value/timezone strings per occurrence, including native message metadata, with aliasing regressions first.

## 3. Documentation and validation

- [x] 3.1 Add Docs/arrowipc.md, both README package rows, docs index and allpkgs registration; update both Unreleased changelogs.
- [x] 3.2 Run build, touched-package tests, vet, lint, govulncheck and strict OpenSpec validation with unchanged go.mod directive.
- [x] 3.3 Manually inspect and sample the three public MaleCNS files via HTTP Range and record observed results in [verification.md](verification.md).
- [x] 3.4 Verify requirements, review diff and documentation, archive in this PR and replace the capability Purpose; validate main specs.
