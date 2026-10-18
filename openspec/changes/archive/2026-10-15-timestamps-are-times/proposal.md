# Proposal: timestamps-are-times

## Why

D-12 and TS-15 of the API review (#220), ported from `0.4` to the 0.3.x line without its breaking part. Every `DataList` and `DataTable` records when it was created and last changed as `time.Now().Unix()`, so two changes within one second look like no change, and `TestDataListGetCreationTimestamp` sleeps for a real second to cross a boundary.

## What Changes

- `DataList` and `DataTable` gain `CreationTime() time.Time` and `LastModifiedTime() time.Time`, at the clock's full resolution. They are stored as Unix nanoseconds.
- `GetCreationTimestamp()` and `GetLastModifiedTimestamp()` are **deprecated**, return Unix seconds as before, and are removed in 0.4.
- `LastModifiedTime` never moves backwards when the wall clock does.
- `IDataList` and `IDataTable` gain the two methods.
- A package clock (`var clock = time.Now`, unexported) lets the tests set time instead of sleeping.
- **Left on `0.4`:** `DataList.IsTheSameAs` comparing data and name only. It is a change of result for any caller who relied on two lists made in different seconds being different. On this line it keeps comparing the two times, in whole Unix seconds, so storing nanoseconds does not make it stricter.

## Capabilities

### New Capabilities

- `object-timestamps`: when a list or table was created and last changed.

### Modified Capabilities

None.

## Impact

- `datalist.go`, `datatable.go`, `timestamps.go` (new), `datalist_sampling.go`, `datatable_sampling.go`, `datatable_filters.go`, `datatable_ccl.go`, `interfaces.go`, and their tests; `ml/models_test.go`'s `bitwiseEqual` skips the two recorded-time fields.
- `Docs/DataList.md`, `Docs/DataTable.md` and both changelogs.
