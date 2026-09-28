## ADDED Requirements

### Requirement: The exported datafetch API names no third-party type

No exported function, method, type, field, variable or constant declared in `datafetch` SHALL name, in its signature or type, a type from a module other than the Go standard library and `github.com/HazelnutParadise/insyra`. The Yahoo Finance methods SHALL convert to go-yfinance's types internally.

#### Scenario: Scanning the package's declarations
- **WHEN** 解析 `datafetch` 所有非測試檔，檢查每個匯出宣告的參數、回傳值、欄位與型別
- **THEN** 沒有任何一個引用 `github.com/wnjoon/go-yfinance` 或其他第三方模組的型別

### Requirement: History parameters are insyra's own and convert field by field

`YFHistoryParams` SHALL be a struct declared in `datafetch` with the fields `Period`, `Interval`, `Start`, `End`, `PrePost`, `AutoAdjust`, `Actions`, `Repair`, `RepairOptions` and `KeepNA`, carrying the meanings and JSON tags they had as `models.HistoryParams`. `RepairOptions` SHALL be a `*YFRepairOptions` with `FixUnitMixups`, `FixZeroes`, `FixSplits`, `FixDividends` and `FixCapitalGains`. `History` SHALL pass every field to go-yfinance unchanged, a nil `RepairOptions` as nil. The two parameter structs SHALL have the same field names as go-yfinance's, checked by a test, so that a go-yfinance upgrade that adds or renames a history parameter fails the tests.

#### Scenario: Every field reaches go-yfinance
- **WHEN** 每個欄位都設為非零值（含 `RepairOptions` 的五個開關）後轉換成 go-yfinance 的參數
- **THEN** 轉換結果的每個欄位與原值相同

#### Scenario: A nil RepairOptions
- **WHEN** `RepairOptions` 為 nil
- **THEN** 轉換結果的 `RepairOptions` 也是 nil

#### Scenario: The field lists match go-yfinance
- **WHEN** 以反射比較 `YFHistoryParams` 與 `models.HistoryParams`、`YFRepairOptions` 與 `models.RepairOptions` 的欄位名稱
- **THEN** 兩邊的欄位名稱集合相同

### Requirement: News takes insyra's own tab type

`News` SHALL take its tab as a `YFNewsTab`. `YFNewsTabNews` (`"news"`), `YFNewsTabAll` (`"all"`) and `YFNewsTabPressReleases` (`"press releases"`) SHALL select what the go-yfinance tabs of the same value select, and the empty tab SHALL select news. Any other value SHALL be refused with an error naming the accepted tabs, before any request is sent.

#### Scenario: An unknown tab
- **WHEN** 以 `YFNewsTab("videos")` 呼叫 `News`
- **THEN** 回傳列出可用分頁的錯誤，且沒有發出請求

#### Scenario: The empty tab
- **WHEN** 以空字串的分頁轉換成 go-yfinance 的分頁
- **THEN** 結果是 news 分頁
