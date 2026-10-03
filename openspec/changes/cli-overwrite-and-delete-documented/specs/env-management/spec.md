## MODIFIED Requirements

### Requirement: Environment delete
系統 SHALL 提供 `env delete <name> [--force]` 命令刪除環境，刪除時 SHALL NOT 出現確認提示。系統 SHALL 拒絕刪除目前使用中的環境，`--force` SHALL NOT 解除這項拒絕。沒有 `--force` 時，系統 SHALL 拒絕刪除 `default` 環境，錯誤訊息 SHALL 說明刪除會失去它的變數與歷史，並說明加上 `--force` 才會刪除。其他未知的旗標 SHALL 回傳錯誤。one-shot 使用時，`--force` SHALL 傳到指令。

#### Scenario: Delete an environment
- **WHEN** 使用者執行 `env delete my-project`，且目前使用的不是 `my-project`
- **THEN** 系統刪除 `~/.insyra/envs/my-project/` 目錄，不詢問確認

#### Scenario: Delete current environment
- **WHEN** 使用者嘗試刪除當前使用的環境，不論有沒有 `--force`
- **THEN** 系統顯示錯誤訊息，禁止刪除當前環境

#### Scenario: Delete default without --force
- **WHEN** 目前使用的環境不是 `default`，使用者執行 `env delete default`
- **THEN** 指令回傳錯誤，訊息提到 `--force`，`default` 環境仍在

#### Scenario: Delete default with --force
- **WHEN** 目前使用的環境不是 `default`，使用者執行 `env delete default --force`（含 one-shot `insyra --env work env delete default --force`）
- **THEN** `default` 環境被刪除
