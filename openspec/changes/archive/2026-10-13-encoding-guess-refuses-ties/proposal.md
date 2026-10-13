# Proposal: encoding-guess-refuses-ties

## Why

The `AGENTS.md` follow-up of 2026-10-13 said encoding detection gave different answers for one short Big5 file. The owner asked to resolve it and ruled on 2026-10-13 (option 甲) after the measurements below, which showed it is much wider than that one file.

- **Root cause.** chardet runs its recognisers in goroutines and sorts their results with the unstable `sort.Sort` (`detector.go:96-113`), so a tie goes to whichever goroutine finished first. Every multi-byte recogniser scores exactly 10 on a sample with at most ten double-byte characters and no invalid ones (`multi_byte.go:49-55`). Big5, GB-18030, Shift_JIS, EUC-JP and EUC-KR therefore tie on any such sample.
- **The commonest real file is affected:** a header such as `名稱,數量,價格` over 400 rows of numbers, 100 detections each.
  - Big5: read as `big5` 48 times and `gb-18030` 52 times.
  - GBK: right 55 times.
  - EUC-KR: right 13 times.
  - Shift_JIS: read as `windows-1252` all 100 times, at confidence 42.
- **What a wrong read looks like:** the header comes back as `ｦWｺﾙ` with a nil error.
- **Common-character tables cannot fix it.** Scoring with chardet's tables could not break these ties: on headers of five or six characters they found nothing, or pointed at the wrong charset.

The ruling is to refuse a guess the input does not support, rather than make it.

## What Changes

- **Auto uses a chardet answer only when it is clear.** The best confidence must be at least 50 out of 100, and no other charset may score the same. Otherwise detection fails with an error wrapping the new `insyra.ErrEncodingAmbiguous`. The message lists the five likeliest charsets with their scores, states the rule, and says to name the encoding.
- **Results are sorted stably** (confidence, then name), so the outcome no longer depends on goroutine order.
- **When chardet names no charset at all, nothing changes:** the read falls back to UTF-8 with a warning (`csv-encoding-integrity`). A lone `0xE9` used to be that case's test input. chardet names it UTF-8 at confidence 10, so it is now refused, and the test uses `0xFF`, which no recogniser names.
- **The CLI's `load` adds a hint** to the error: `add encoding <name> to the load command`.
- **Documented in** `Docs/utils.md`, `Docs/csvxl.md`, `Docs/DataTable.md` and `Docs/cli-dsl.md`. They state the rule, the measured scores and the Shift_JIS weak spot.

## Measured effect (before → after)

- **Wrong or random reads:** all but one now fail with the error.
  - Refused: the Big5, GBK, EUC-KR and Shift_JIS headers over numbers, the 24-byte Big5 file, both short windows-1251 files (read as Hebrew before), and ten Shift_JIS rows (read as windows-1252 at 49).
  - Still wrong: one Shift_JIS header over 300 rows of numbers, scored 60 as windows-1252, is read as windows-1252. It is documented and recorded as a follow-up.
- **Correct reads:**
  - 5 of 7 stay automatic: Big5, GBK and EUC-KR text, and longer Latin-1.
  - Two very short Latin-1 files (confidence 27 and 35) now need `encoding iso-8859-1` or `windows-1252`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `core-encoding-detection`: detection refuses an unclear guess, and its answer does not depend on scheduling.

## Impact

- **Code:** `utils.go` (`DetectEncoding`, the new `ErrEncodingAmbiguous`) and `internal/dsl/commands/load.go`.
- **Tests:** `encoding_guess_test.go`, `batch6_test.go`, `csvxl/read_csv_encoding_test.go` and `internal/dsl/commands/load_save_test.go`.
- **Every reader that detects is affected:** `ReadCSVFile`, `ReadCSVString`, `ReadCSV`, `StreamCSV`, `DetectEncoding`, and the `csvxl` readers and converters with `Auto`.
- **Breaking:** an input whose encoding is unclear is now an error where it used to be read with a guess.
- **Docs and records:** both changelogs (`### Core`), `AGENTS.md` (the follow-up is replaced by one for the Shift_JIS miss), `delivery-status.md`.
