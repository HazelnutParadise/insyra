# Proposal: gmaps-search-restored

## Why

[#249](https://github.com/HazelnutParadise/insyra/issues/249) (DF-1, SEC-14). The Google Maps crawler has carried a `FIXME: this crawler doesn't work anymore` header while staying public. Measured on 2026-09-13:

| Call | Result |
| --- | --- |
| `Search("鼎泰豐")`, `Search("Din Tai Fung Taipei")` | 0 stores, no warning. Google now answers the page the crawler reads with 247 KB that holds no store IDs. |
| `GetReviews` on a real store ID | HTTP 403 from `listugcposts`, with the old and a current browser user agent, and with a fresh session token |
| `GoogleMapsStores()` | downloads the URLs and request headers from `raw.githubusercontent.com/TimLai666/google-maps-store-review-crawler` on every call, so whoever controls that repository controls where the crawler sends requests and what it sends |

The owner asked for what can be fixed now to be fixed, and for the reviews to keep being explored.

The Maps web page itself lists search results by requesting `/search?tbm=map` with a request descriptor (`pb`). Replayed over plain HTTP that request returns the stores with their feature IDs and names. The page's descriptor is 1,643 characters; dropping fields one at a time and comparing results for 鼎泰豐 (11 stores) and 星巴克 (20) showed a 99-character descriptor returns the same stores. The query is taken from `q`, not from the descriptor, and no session token is needed.

Reading the rest of the file turned up three more defects: no request has a timeout, `GetReviews` prints progress to standard output, and `MaxWaitingInterval_Milliseconds: 1000` reaches `rand.IntN(0)`, which panics.

## What Changes

- **`Search` reads the result list the Maps page requests.** One request returns up to 20 stores with names, in Google's order, with repeats and entries without a feature ID dropped. It no longer fetches one page per store to scrape its name. When nothing comes back it warns that the response format may have changed, instead of returning nothing silently.
- **No remote configuration.** The endpoints and the user agent are constants. `GoogleMapsStores()` needs no network access and never returns nil.
- Every request goes through one client with a 30-second timeout and reads at most 64 MiB.
- `GetReviews` logs progress at debug level, no longer panics at a waiting interval of 1000, and treats a zero `SortBy` or `MaxWaitingInterval_Milliseconds` as its default without a warning.
- `GetReviews` still calls `listugcposts`, which Google refuses. The doc comment and `Docs/datafetch.md` say so. Restoring reviews is not part of this change; the exploration continues on #249.
- A live test behind the `gmaps_live` build tag checks search against Google.

## Capabilities

### New Capabilities

- `datafetch-gmaps-crawler`: what the Google Maps crawler sends, how it reads search results, and how it reports failure.

### Modified Capabilities

(none)

## Impact

- `Search` returns stores again. Names now come from the search response rather than each store's page.
- `GoogleMapsStores()` no longer returns nil; existing nil checks keep compiling and never fire.
- Code that relied on `GetReviews` printing progress to standard output sees nothing unless the log level is Debug.
- `GetReviews` still returns nil with a warning until reviews are restored.
