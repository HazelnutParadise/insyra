# Design: datafetch-own-yfinance-types

## Same shape, own declaration

`YFHistoryParams` keeps go-yfinance's field names, pointer dates and JSON tags. The point of the change is that insyra owns the declaration, not that the parameters are redesigned: keeping the shape means every composite literal written against the alias still compiles, and a `YFHistoryParams` marshalled to JSON by a caller comes out as it did. A redesign (a `time.Time` start instead of a pointer, a period type instead of a string) would break those callers for no gain the finding asked for.

## Catching a go-yfinance upgrade

With an alias, an upgrade of go-yfinance changed insyra's API. With a copy, the risk turns around: go-yfinance could add a history parameter that insyra never passes on. The field-name comparison test makes that visible at the upgrade, where the person bumping the dependency decides whether to expose the new field. The conversion itself is written field by field, so a renamed field fails to compile.

## The news tab

The three accepted values are go-yfinance's own. go-yfinance treats any unknown tab as news; insyra refuses one instead, because the type is now insyra's and a typo such as `"press-releases"` should not silently fetch something else. The empty tab keeps meaning news, which is what the CLI passes.
