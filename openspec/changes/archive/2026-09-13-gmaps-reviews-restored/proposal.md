# Proposal: gmaps-reviews-restored

## Why

[#249](https://github.com/HazelnutParadise/insyra/issues/249) (DF-1). `gmaps-search-restored` brought `Search` back but left `GetReviews` failing, and the owner asked for the reviews to be restored too. Measured on 2026-09-13:

| Route | Signed out |
| --- | --- |
| `GET /maps/rpc/listugcposts` (what the crawler sends, ported from the 2024 article it credits) | HTTP 403 |
| Google Maps place page, reviews tab | 5 reviews. Scrolling the review list to the bottom loads no more; the page says "Google 地圖目前只顯示部分內容"; filtering by a topic raises a sign-in prompt |
| Maps `batchexecute` rpc `qv9Egd` | 5 reviews per sort order, no next-page token, and a session token that works roughly one time in three |
| `GET /async/reviewDialog` (the route scraping guides describe) | HTTP 404 on google.com.tw and google.com |
| Google Search results, "查看所有 Google 評論" | 10 reviews, and scrolling loads 10 more |

The Search review window loads each page with `GET /httpservice/web/PrivateLocalSearchUiDataService/GetLocalBoqProxy`, whose `reqpld` parameter carries the store's feature ID, the sort order, a page size of 10 and a next-page token. Replayed over plain HTTP with no cookies it returned the same pages as the browser: six consecutive pages gave 60 distinct reviews and a token every time. The sort field takes 1 to 4, and measured on 鼎泰豐 板橋店, 2 returned the newest first, 3 ten 5-star reviews and 4 ten 1-star reviews, which are the values `SortByRelevance`, `SortByNewest`, `SortByHighestRating` and `SortByLowestRating` already hold. A second store (星巴克 中和景安門市) answered the same way.

This is the same mechanism the crawler was built on, a page token that walks the review list 10 at a time, on the endpoint Google now uses for it, so `GetReviews` keeps its signature and its meaning.

## What Changes

- **`GetReviews` reads the Search review pages.** Each request asks for 10 reviews in the requested sort order and carries the previous page's token; it stops after `pageCount` pages, or when a page has no token, with `pageCount` 0 meaning every page. The random wait between pages is unchanged.
- **Fields.** `Reviewer`, `ReviewerID` (the number in the reviewer's contributions link), `ReviewTime` (Google's relative time), `Rating` and `Content` come from the new response. `ReviewDate` is the review's timestamp as a UTC `YYYY-MM-DD`. `Content` has `<br>` turned into line breaks and HTML entities decoded. `ReviewerState` and `ReviewerLevel` stay empty: the response no longer carries a reviewer's status line or guide level.
- A page whose response has no review block is reported as a warning saying the format may have changed, and `GetReviews` returns nil. A page with an empty review list is not an error.
- The doc comment and `Docs/datafetch.md` no longer say reviews are unavailable.
- A live test behind the `gmaps_live` build tag reads two pages of newest reviews.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datafetch-gmaps-crawler`: adds what `GetReviews` requests and how it reads a review page.

## Impact

- `GetReviews` returns reviews again instead of nil.
- `ReviewerState` is always empty and `ReviewerLevel` always 0; the two columns stay in `ToDataTable` so existing column lookups keep working.
- `ReviewDate` is computed from a timestamp in UTC, so a review posted just after midnight in Taiwan carries the previous day's date.
- Like the rest of the crawler, this reads an undocumented Google endpoint that can change without notice.
