# Tasks: cli-result-variables-named

## 1. 指令輸出

- [x] 1.1 `cli/commands/result_variables_test.go`：`TestCommandsNameEveryVariableTheyStore` 確認各指令執行後，輸出列出它另存的每個變數名稱
- [x] 1.2 `cli/commands/registry.go` 新增 `alsoStored`；`clustering.go`（kmeans、dbscan、silhouette）、`pca.go`、`knn.go`（knn_classify、knn_neighbors）、`quant.go`（quant factor、quant portfolio）的成功訊息改用它列出另存的變數

## 2. 紀錄

- [x] 2.1 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `### CLI` 段落末尾新增條目
