# Tasks: cli-result-variables-named

## 1. 指令輸出

- [x] 1.1 先寫會失敗的測試：`kmeans`、`dbscan`、`silhouette`、`pca`、`corrmatrix`、`knn_classify`、`knn_neighbors`、`quant factor`、`quant portfolio` 執行後新增的每個變數名稱都完整出現在輸出中；在舊程式上記錄失敗（`corrmatrix` 以外全部失敗）
- [x] 1.2 `helpers.go` 新增 `alsoStored`，各指令的成功訊息列出另存的變數；1.1 通過

## 2. 驗證

- [x] 2.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）通過
- [x] 2.2 以建好的 CLI（`HOME` 指向暫存目錄）實測 `kmeans`、`pca`、`quant portfolio` 的輸出，以及 `hclust` 樹跨 one-shot 指令後 `cutree` 可用、`regression` 結果被警告不存入

## 3. 文件

- [x] 3.1 `Docs/cli-dsl.md`：Extra Result Variables 表格補上 `quant factor` 與 `quant portfolio`，說明另存變數會取代同名變數、成功訊息會列出名稱，並說明 `hclust` 樹與 `regression` 結果的存活期與讀取指令
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `### CLI` 末尾新增條目
- [x] 3.3 `api-review.md` 的 CLI-17 標為已修正；`delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 3.4 檢查 `skills/use-insyra-cli/`：已說明 regression 結果不存入環境，不列指令，本變更不需修改
- [x] 3.5 `openspec validate cli-result-variables-named --strict` 通過
