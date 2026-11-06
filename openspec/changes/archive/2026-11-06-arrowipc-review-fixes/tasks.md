## 1. Tests

- [x] 1.1 70,000 batches inspect and stream; a 288 MB batch streams with `MaxBatchBytes` raised and fails under the default (large test, opt-in)
- [x] 1.2 Duration keeps its unit; an overflowing count is an error
- [x] 1.3 A wrong size wraps `io.ErrUnexpectedEOF`; repeated field names are renamed
- [x] 1.4 `FileInfo` tests compare `Columns` and `Metadata`; the Parquet parity test compares Duration as `time.Duration`
- [x] 1.5 arrow-go version pin

## 2. Implementation

- [x] 2.1 Arrow body and metadata limits from the options; drop the block-count cap
- [x] 2.2 `FileInfo`/`ColumnInfo`; Duration; short reads; renaming
- [x] 2.3 `dev`: `iter.Seq2`, `[]any` selectors, root `insyra.Name`

## 3. Documentation

- [x] 3.1 `Docs/arrowipc.md`, both changelogs, `delivery-status.md`
