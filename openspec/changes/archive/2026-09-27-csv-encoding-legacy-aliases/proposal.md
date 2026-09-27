# Proposal

## Why

v0.3.2 matched an encoding name by substring, so `utf-8-sig`, `utf-8-bom`, `big5-hkscs`, `x-gbk` and `gb_2312-80` all read. This line looks the name up in a table and refuses anything the table does not hold, which is right for a name nothing decodes, but these names do have a decoder: they are aliases of UTF-8, Big5 and GBK. Measured on 2026-09-27, every one of them fails with `unsupported encoding`, and the BREAKING changelog entry for the refusal does not say that names which used to read stop reading. `utf-8-sig` is the name Python writes a UTF-8 file with a byte-order mark under, so it is a name a user is likely to pass.

## What Changes

- The decoder table gains the aliases v0.3.2 read and their standard labels: `utf-8-sig` and `utf-8-bom` read as UTF-8 with a leading byte-order mark removed; `big5-hkscs`, `csbig5`, `cn-big5` and `x-x-big5` read as Big5; `x-gbk`, `gb_2312-80`, `csgb2312`, `csiso58gb231280`, `chinese` and `iso-ir-58` read as GBK (decoded through GB18030, like the table's other GB names).
- A name outside the table is still refused with the unsupported-encoding error. Nothing falls back to substring matching or to reading raw bytes.

## Capabilities

### New Capabilities

### Modified Capabilities
- `csv-encoding-integrity`: the supported set names these aliases.

## Impact

- `internal/csv/decoder.go`: the `decoders` table.
- Every CSV reader that takes an encoding name (`ReadCSV_File`, `csvxl.ReadCsvToString`, the CSV-to-Excel converters), since they share the table. The unsupported-encoding error lists the new names.
- `Docs/csvxl.md` and both changelogs.
