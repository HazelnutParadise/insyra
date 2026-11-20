# AGENTS.md (parquet/)

Rules for working in `parquet/`. The top-level [AGENTS.md](/AGENTS.md) covers the wider repo.

## Known pitfalls

- A test of row-group-dependent behaviour (type settling, held runs) in `ApplyCCL` or the readers writes its fixture with `WriteOptions{RowGroupSize: 1000}` or similar; `parquet.Write` defaults to 1,048,576 rows per group, so a small fixture is a single row group and never crosses a boundary.
