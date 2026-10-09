## ADDED Requirements

### Requirement: A command that stores several variables names each one
在 `as <var>`（或 `$result`）之外另存變數的指令 SHALL 在輸出中列出它存入的每一個變數名稱，因為每個另存的變數都會取代同名的既有變數。至少包含：`kmeans`（`<var>_centers`、`<var>_size`、`<var>_withinss`、`<var>_totss`、`<var>_totwithinss`、`<var>_betweenss`、`<var>_iter`、`<var>_ifault`）、`dbscan`（`<var>_isseed`）、`silhouette`（`<var>_avg`）、`pca`（`<var>_eigenvalues`、`<var>_explained_variance`）、`corrmatrix`（`<var>_p`）、`knn_classify`（`<var>_classes`、`<var>_probs`）、`knn_neighbors`（`<var>_distances`）、`quant factor`（`<var>_alpha`）與 `quant portfolio`（`<var>_stats`）。`Docs/cli-dsl.md` SHALL 列出每個指令另存的變數，並說明 `hclust` 樹與 `regression` 結果能存活多久、哪些指令讀得到它們。

#### Scenario: kmeans names all nine variables
- **WHEN** 使用者執行 `kmeans t 2 as km`
- **THEN** 輸出列出 `km` 與八個 `km_*` 變數的名稱，且這九個變數都已存入

#### Scenario: quant factor names its alpha table
- **WHEN** 使用者執行 `quant factor a f as fm`
- **THEN** 輸出列出 `fm` 與 `fm_alpha`
