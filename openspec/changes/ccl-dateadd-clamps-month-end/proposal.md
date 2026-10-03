# Proposal: ccl-dateadd-clamps-month-end

## Why

`DATEADD` by months or years went through Go's `time.AddDate`, which carries a day the target month lacks into the next month: `DATEADD('2024-01-31', 1, 'month')` gave `2024-03-02` and `DATEADD('2024-02-29', 1, 'year')` gave `2025-03-01`. Excel's `EDATE` gives `2024-02-29`, and so does pandas: run on 2026-10-03 with pandas 2.3.3, `Timestamp('2024-01-31') + DateOffset(months=1)` is `2024-02-29`. Anyone adding a month to a month-end date expects the end of the next month, not a date in the month after. Issue #367 (CCL-33); its overflow half was fixed by `ccl-portable-integer-arguments`.

## What Changes

- **BREAKING**: a `month` or `year` shift keeps the day of the month, and stops at the target month's last day when that month is shorter. The time of day and the time zone are kept.
- Shifts by days, hours, minutes and seconds are unchanged, and the range checks from `ccl-portable-integer-arguments` stay as they are.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ccl-type-semantics`: DATEADD by months stops at the month's end.

## Impact

- `internal/ccl/stdlib_datetime.go`: `addMonths`, used by the `month` and `year` units.
- Results change only where the target month is shorter than the start date's day: day 29, 30 or 31 shifted into a shorter month.
- `Docs/CCL.md`, both changelogs, `api-review.md`, `delivery-status.md`.
