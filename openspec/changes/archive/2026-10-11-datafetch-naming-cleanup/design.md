# Design: datafetch-naming-cleanup

## Which synonym stays

go-yfinance's own documentation calls `"annual"` the frequency and `"yearly"` an alias kept for Python yfinance compatibility, and `Docs/datafetch.md` already lists `YFPeriodAnnual` as the default. `YFPeriodAnnual` stays. Making `YFPeriodYearly` equal to `"annual"` would change the label of every table a caller fetches with it, so it keeps `"yearly"` until it is removed in 0.4, as the #211 ruling requires of a deprecated name.

## Two waiting fields

The new field reuses every rule of the old one, including that a value under one second is replaced by the default with a warning rather than refused: changing that is a separate question from the name and the type. When both fields are set, one would have to win silently, and the library's rule for a trailing options value is that more than one is an error. `GetReviews` reports every error as a warning and a `nil` result, so a conflict is reported the same way, before any request.

The wait is drawn in nanoseconds between one second and the limit, so a limit such as 1.5 seconds is honoured as written rather than truncated to milliseconds.

## Constant names

The new sort-order names follow the package's pattern exactly, the type's name followed by the value, even though `GoogleMapsStoreReviewSortBy` makes them long: a shorter prefix would be a third pattern next to `TWMarketXxx` and `YFPeriodXxx`.