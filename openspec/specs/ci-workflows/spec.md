# ci-workflows Specification

## Purpose
Defines what CI checks and where it runs: every Go file is gofmt-formatted, the Test workflow vets on every OS, runs the race detector on ubuntu and reports coverage from another leg, every workflow runs with least-privilege tokens, and every workflow that runs on dev also runs on 0.4.

## Requirements

### Requirement: Formatting is checked

Every Go file in the repository SHALL be formatted as `gofmt` formats it, and the lint job SHALL fail on a file that is not.

#### Scenario: An unformatted file
- **WHEN** 有任何 Go 檔案是 gofmt 會改動的
- **THEN** `golangci-lint run` 回報 gofmt 問題，CI 的 lint 工作失敗

### Requirement: The test job vets, runs the race detector and reports coverage

The Test workflow SHALL run `go vet ./...` on every OS leg and SHALL run the suite with `-race` on the ubuntu leg. It SHALL report the total coverage from a leg that does not run the race detector, and one leg failing SHALL NOT cancel the others.

#### Scenario: A data race
- **WHEN** 某個測試觸發 data race
- **THEN** ubuntu 那一條的測試失敗，另外兩條照樣跑完

### Requirement: Workflows run with least privilege

Every workflow SHALL declare its token permissions, and a workflow that only reads the repository SHALL declare `contents: read`. A checkout SHALL NOT persist credentials when the job authenticates another way.

#### Scenario: deploy-docs
- **WHEN** deploy-docs 部署文件
- **THEN** checkout 不保留憑證，推送改用 actions-gh-pages 自己的 token

### Requirement: Every line of development runs CI

Every workflow that runs on `dev` SHALL also run on `0.4`, the branch that carries the API review.

#### Scenario: A push to 0.4
- **WHEN** 推送到 `0.4`
- **THEN** Test、GolangCI-Lint、Govulncheck 與參考對照的 workflow 都會執行
