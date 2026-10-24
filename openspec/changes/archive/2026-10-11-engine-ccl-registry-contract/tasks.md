# Tasks: engine-ccl-registry-contract

## 1. 測試先行

- [x] 1.1 `engine/ccl/registry_contract_test.go`：`RegisterSequenceFunction` 註冊的函式能被求值、兩個 no-op 有 `Deprecated:`、三個 `Register*` 的說明寫明登錄表行為；在舊程式上確認失敗（編譯不過）

## 2. 實作

- [x] 2.1 `engine/ccl/ccl.go`：加 `SeqFunc`、`RegisterSequenceFunction`；兩個 no-op 標 Deprecated；套件說明與每個 `Register*` 寫明登錄表行為

## 3. 文件與紀錄

- [x] 3.1 `Docs/CCL.md` Custom Functions、`engine/README.md` CCL 章節：三種註冊函式與登錄表行為，拿掉過時的 `sync.Once` 建議
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### Core` 段落末尾新增條目
- [x] 3.3 skills：不需要修改，skills 不列函式，文件位置也沒變

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate engine-ccl-registry-contract --strict`
