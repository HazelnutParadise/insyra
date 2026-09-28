# Proposal: show-whole-up-to-60-rows

## Why

#236 (E-8) asked whether `Show()` should truncate long tables. It already did: since 323ec68d (2025-05-19) a view with no range printed a table or list of more than 25 rows as its first 20 and last 5. The owner ruled on 2026-09-28 that a view of up to 60 rows prints whole and a longer one keeps the first 20 and last 5. For comparison, measured with pandas 2.3.3: `display.max_rows` is 60 and `display.min_rows` is 10, so pandas prints up to 60 rows whole and 5 plus 5 past that.

## What Changes

- **BREAKING (display only)**: `Show`, and `ShowRange` and `ShowTypesRange` with no range, on DataTable and DataList alike, print up to 60 rows whole. Past 60 they print the first 20 and last 5 with `...` between, as before. One constant, `showWholeUpTo`, holds the threshold for all four views; it was the literal 25 in three places.
- The doc comments and `Docs/DataTable.md` and `Docs/DataList.md` state the rule. The comments still said a view with no range showed every row.
- An explicit range still prints every row in it, as before.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `show-range`: adds "A view with no range prints up to 60 rows whole"; the default-view scenario of "A table view is sized by the rows it prints" moves from a 30-row table, which now prints whole, to a 70-row one.

## Impact

- `show.go`; tests in `show_layout_test.go`.
- `Docs/DataTable.md`, `Docs/DataList.md`, both changelogs (a new entry, and the sentence of the `show-measures-shown-rows` entry that stated the 25-row rule removed), `api-review.md`, `delivery-status.md`.
- The agent skills say nothing about how many rows a view prints and are unchanged.
