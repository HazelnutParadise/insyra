## ADDED Requirements

### Requirement: A doubled quote inside a literal is one quote

A string literal SHALL be enclosed in single or double quotes. Inside it, the enclosing quote character written twice SHALL stand for one such character, and the other quote character SHALL stand for itself. A bracketed column name SHALL follow the same rule. A backslash SHALL be an ordinary character. Splitting a statement-mode script into statements SHALL treat a doubled quote as part of the literal, so a `;` or a line break inside the literal does not end the statement.

#### Scenario: Single-quoted literal holding a single quote
- **WHEN** 求值 `'it''s'`
- **THEN** 得到字串 `it's`

#### Scenario: Double-quoted literal holding double quotes
- **WHEN** 求值 `"say ""hi"""`
- **THEN** 得到字串 `say "hi"`

#### Scenario: A literal that is one quote
- **WHEN** 求值 `''''`
- **THEN** 得到字串 `'`

#### Scenario: Bracketed column name with a quote
- **WHEN** 表格有名為 `O'Brien` 的欄，求值 `['O''Brien']`
- **THEN** 得到該欄的值

#### Scenario: A backslash does not escape
- **WHEN** 編譯 `'it\'s'`
- **THEN** 回傳 `unclosed string` 錯誤

#### Scenario: Statement splitting keeps the literal whole
- **WHEN** 以 statement mode 編譯 `NEW('x') = 'a;b''c` 換行 `d'` 換行 `NEW('it''s') = 1`
- **THEN** 得到兩個敘述，第一個的字串是 `a;b'c` 換行 `d`，第二個建立名為 `it's` 的欄
