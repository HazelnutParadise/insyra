# cli-env-name Specification

## Purpose
An environment name is joined onto the environments directory, so a name whose joined path would not stay strictly inside that directory is refused, closing the path-traversal hole without restricting the characters existing names use.

## Requirements
### Requirement: Environment names cannot escape the envs directory

`Manager.ResolveEnvPath`（因此 `Create`／`Open`／`Delete`／`Rename`／`Clear`／`Export`／`Import`）SHALL 拒絕空白名稱、絕對路徑或以分隔符開頭的名稱，以及清理後路徑不嚴格位於環境目錄內的名稱（經 `..` 跳出，或解析成環境目錄本身）。其他名稱（含空格、非 ASCII 字元或以 `.` 開頭）SHALL 照常接受。

#### Scenario: Traversal is refused
- **WHEN** `Create("../outside")` 或 `Create("a/../../outside")`
- **THEN** 回傳錯誤，環境目錄外沒有建立任何資料夾

#### Scenario: Absolute name is refused
- **WHEN** `Create("/abs")`
- **THEN** 回傳錯誤

#### Scenario: Name resolving to the directory itself is refused
- **WHEN** `Create("a/..")`
- **THEN** 回傳錯誤

#### Scenario: Existing name shapes still work
- **WHEN** `Create("my env")`、`Create("中文")`、`Create(".hidden")`
- **THEN** 皆成功，路徑位於環境目錄內
