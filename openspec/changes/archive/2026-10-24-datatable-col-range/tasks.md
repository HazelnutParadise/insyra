# Tasks

- [x] Tests first: `datatable_colrange_test.go` (both ends, spellings, nil and negative bounds, errors, padding, ownership, empty table, concurrency, interface).
- [x] `ColRange` in `datatable_filters.go` beside `SliceCols`, sharing a private copy helper; add it to `IDataTable`.
- [x] `Docs/DataTable.md`: a `### ColRange` section and a paragraph in Slicing.
- [x] Entries under `## Unreleased` / `### Core` in both CHANGELOGs.
- [x] `openspec validate datatable-col-range --strict` passes.
