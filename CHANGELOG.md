# Changelog

Changes that affect people using Insyra, grouped by package the same way release notes are. `## Unreleased` holds what will go into the next release.

v0.3.0 and everything before it is not repeated here — see [GitHub Releases](https://github.com/HazelnutParadise/insyra/releases).

繁體中文：[CHANGELOG_TW.md](CHANGELOG_TW.md)

## Unreleased

### Core

- Added `CSVReadOptions` together with `ReadCSV_FileWithOptions` and `ReadCSV_StringWithOptions`. Setting `RawStrings` keeps every cell as its original string and skips column-level type inference, so values like stock IDs no longer lose their leading zeros and empty cells stay `""` instead of becoming NaN. `ReadCSV_File` and `ReadCSV_String` keep their existing signatures and behavior.

### `isr`

- `CSV_inOpts` gains a `RawStrings` field, which `DT.From` passes through to the reader.

### CLI

- `load <file.csv>` accepts `infer true|false`, defaulting to `true`. Passing `infer false` loads every cell as a raw string. The option is rejected for JSON and Excel files.
