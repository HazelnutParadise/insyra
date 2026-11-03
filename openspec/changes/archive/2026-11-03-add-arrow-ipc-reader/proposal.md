## Why

Issue #377 needs schema discovery and bounded reads of scientific Feather V2 datasets without loading an entire file or losing integer IDs. The existing Arrow v18 dependency supports the IPC file format used by Feather V2.

## What Changes

- Add read-only `arrowipc` with metadata-only inspection from paths and `io.ReaderAt`, and Parquet-shaped `Stream` / `StreamFrom` APIs, with row and decoded-byte bounds.
- Preserve Parquet cell mappings, including exact integer widths, nulls, binary, dates and decimals. Decode dictionaries to values and explicitly reject unsupported selected types, including lists.
- Add generated fixtures, offline range-read checks, documentation and bilingual changelog entries on dev and 0.4.

## Capabilities

### New Capabilities
- `arrow-ipc-reading`: schema inspection and bounded, cancellable IPC file / Feather V2 batch reading.

### Modified Capabilities
None.

## Impact

New `arrowipc/` package, `Docs/arrowipc.md`, both README and changelog entry points, docs index and allpkgs registration. No dependency, Go directive, existing API, CLI or writer changes. Manual MaleCNS range probes are evidence only and never CI inputs.

## Delivery

The owner explicitly requires this change to be archived in each PR before merge. Open one PR to dev and one to 0.4. The owner has now authorized merging both after all CI is green, every review finding is addressed and a fresh review finds no new issues.
