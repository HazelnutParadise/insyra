# Tasks: cli-overwrite-and-delete-documented

## 1. `env delete default`

- [x] 1.1 先寫會失敗的測試：沒有 `--force` 時 `env delete default` 回傳提到 `--force` 的錯誤且環境仍在；`--force` 時刪除；`--force` 不解除刪除使用中環境的拒絕；未知旗標回傳錯誤；one-shot `--env work env delete default --force` 刪除 `default`；`help save`、`help plot`、`help env` 說明取代與刪除；在舊程式上記錄失敗
- [x] 1.2 `env.go` 解析 `env delete <name> [--force]` 並拒絕沒有 `--force` 的 `default`；`CommandFlag.Form` 可用 `|` 列出多個字（`cli/commands/cobra.go` 比對），`env` 的 `--force` 宣告為 `import|delete`，one-shot 時傳給 `env delete`；`cli/AGENTS.md` 說明；`env`、`save`、`plot` 的 `Forms` 說明取代與刪除；1.1 通過

- [x] 1.3 依 review 結果補強：刪除與重新命名比對 `Lstat(目標)` 與 `Stat(目前環境)`，清空與匯入兩邊都跟著連結；`env clear`、`env import`、`env rename` 與 `env list` 都以目錄認出目前環境；先寫在舊程式上失敗的測試（大小寫與符號連結），再修到通過

## 2. 驗證

- [x] 2.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）通過
- [x] 2.2 以建好的 CLI（`HOME` 指向暫存目錄）實測 `env delete default` 有無 `--force`、`save` 兩次、`plot` 不帶 `save`、`env export` 兩次、`env clear` 有無 `--keep-history`

## 3. 文件

- [x] 3.1 `Docs/cli-dsl.md`：Environment Model 新增會取代或刪除資料的指令清單，`save` 與 Plot Output 段落說明取代既有檔案
- [x] 3.2 `skills/use-insyra-cli/SKILL.md`：隔離工作的原則把 `env delete`、`env clear` 與覆寫檔案列為破壞性操作
- [x] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `### CLI` 末尾新增條目（標 BREAKING：`env delete default` 需要 `--force`）
- [x] 3.4 `api-review.md` 的 CLI-11 標為已修正；`delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 3.5 `openspec validate cli-overwrite-and-delete-documented --strict` 通過
