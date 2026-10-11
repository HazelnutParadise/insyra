# Tasks: cli-rank-tests

## 1. 指令

- [x] 1.1 測試：`wilcoxon single`、`wilcoxon paired`、`mannwhitney`、`kruskal` 的輸出等於程式庫以同樣資料與對立假設呼叫的結果；省略對立假設等於雙尾；拼錯的對立假設、參數不足、多餘參數與程式庫錯誤都回傳錯誤；one-shot 模式支援負數 mu
- [x] 1.2 `cli/commands/nonparam.go` 註冊並實作三個指令，設定 `DisableFlagParsing` 支援負數 mu，並嚴格檢查參數長度

## 2. 驗證

- [x] 2.1 驗證通過：`go test ./cli/commands/...` 通過

## 3. 文件

- [x] 3.1 `Docs/cli-dsl.md`：指令索引新增三列，指令分組加入三個指令，新增說明段落
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `### CLI` 末尾新增條目
