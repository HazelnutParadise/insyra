# Proposal: gmaps-review-extra-fields

## Why

`gmaps-reviews-restored` moved `GetReviews` to the review pages of Google Search's review window. Those pages carry four facts about each review that `GoogleMapsStoreReview` has no field for, and the owner asked for them to be added:

| Fact | Position in a review record | Checked against |
| --- | --- | --- |
| The review's ID | `[5]` | the review's permalink and report link, which embed the same ID |
| The review's language code | `[26]` | 20 saved reviews, all `zh-Hant`. Empty on a review with only a star rating: 5 of 20 live reviews had neither text nor language |
| How many reviews the reviewer has written | `[3][3]` | the window's "18 則評論" and "145 則評論" for the same reviewers |
| How many photos the reviewer has posted | `[3][4]` | the window's "173 張相片" and "727 張相片" |

`ReviewerState` and `ReviewerLevel` are empty since that change, so the counts are the only information about a reviewer beyond name and ID. A review ID also lets a caller drop repeats when merging pages fetched in different sort orders.

## What Changes

- `GoogleMapsStoreReview` gains `ReviewID`, `Language`, `ReviewerReviewCount` and `ReviewerPhotoCount`, and `GetReviews` fills them.
- `ToDataTable` adds a column for each, named after the field.
- `Docs/datafetch.md` lists the fields and columns.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datafetch-gmaps-crawler`: `GetReviews` reports the review ID, language and the reviewer's counts.

## Impact

- Additive. Code that builds a `GoogleMapsStoreReview` with positional fields stops compiling; keyed literals and field access are unaffected.
- `ToDataTable` returns four more columns.
