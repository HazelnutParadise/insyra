# Spec Delta

## ADDED Requirements

### Requirement: Common aliases of a supported charset are accepted

編碼名稱是受支援字元集的常見別名時，讀取 SHALL 以該字元集解碼，SHALL NOT 回傳 unsupported-encoding 錯誤。`utf-8-sig` 與 `utf-8-bom` SHALL 讀成 UTF-8 並去掉開頭的位元組順序標記；`big5-hkscs`、`csbig5`、`cn-big5`、`x-x-big5` SHALL 讀成 Big5；`x-gbk`、`gb_2312-80`、`csgb2312`、`csiso58gb231280`、`chinese`、`iso-ir-58` SHALL 讀成 GBK。名稱中的大小寫與分隔符號 SHALL NOT 影響比對。不屬於任何受支援字元集的名稱 SHALL 仍回傳錯誤。

#### Scenario: A file written as utf-8-sig
- **WHEN** 以 `utf-8-sig` 讀取開頭有 UTF-8 位元組順序標記的 CSV
- **THEN** 讀取成功，內容不含位元組順序標記

#### Scenario: A Big5 file named big5-hkscs
- **WHEN** 以 `BIG5-HKSCS` 讀取 Big5 編碼的 CSV
- **THEN** 儲存格是正確解碼的 UTF-8 文字

#### Scenario: A name no charset owns
- **WHEN** 以 `klingon-1` 讀取
- **THEN** 回傳 unsupported-encoding 錯誤
