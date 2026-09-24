# Tasks: ci-hygiene

## 1. Formatting

- [x] 1.1 `.golangci.yml` 開啟 `gofmt` formatter
- [x] 1.2 排版 24 個檔案，包含 `stats/internal/fa`（只動註解縮排、空行與 import 順序，`git diff -w` 確認沒有程式碼行被改）

## 2. Test workflow

- [x] 2.1 每個 OS 跑 `go vet ./...`
- [x] 2.2 ubuntu 那一條加 `-race`；總覆蓋率改在 macOS 那一條產生、寫進 job summary（race 與 atomic 覆蓋率疊在一起時，clustering 超過 10 分鐘逾時；分開各自不到 30 秒）
- [x] 2.3 矩陣改 `fail-fast: false`（第一次跑時 windows 的偶發下載錯誤把 ubuntu 取消了）

## 3. Permissions and branches

- [x] 3.1 每個 workflow 宣告 `permissions`，只讀的用 `contents: read`
- [x] 3.2 deploy-docs 改 `persist-credentials: false`
- [x] 3.3 在 `dev` 會跑的 workflow，`0.4` 也會跑

## 4. Found by the first run on 0.4

- [x] 4.1 兩個斷言 POSIX 權限位元的測試，在 windows 上跳過該斷言
- [x] 4.2 amd64 上的 CCL 大數轉換 bug，另開 `ccl-portable-integer-arguments` 修正

## 5. Docs and ledger

- [x] 5.1 `AGENTS.md` 的 lint 指令說明
- [x] 5.2 `api-review.md`：RP-3、RP-4、RP-5 標為已修正
- [x] 5.3 `delivery-status.md`

## 6. Verification

- [x] 6.1 本機：`gofmt -l .` 為空、`golangci-lint run` 0 issues、`go vet ./...` 乾淨、`go test ./...` 全綠、`go test -race ./...` 全綠（97 秒）
- [x] 6.2 推送後，GitHub 上六個 workflow 全部通過，Test 的三個 OS 都跑完（run 34630799983 等六個，commit 1f91e40）
- [x] 6.3 `openspec validate ci-hygiene --strict`
