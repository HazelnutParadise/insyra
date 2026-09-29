## ADDED Requirements

### Requirement: One options type for trees

`DecisionTreeOptions` SHALL be the options type both `FitDecisionTreeClassifier` and `FitDecisionTreeRegressor` take. `DecisionTreeClassifierOptions` and `DecisionTreeRegressorOptions` SHALL remain for one release as Deprecated aliases of `DecisionTreeOptions`, each doc comment naming it.

#### Scenario: The deprecated aliases are the same type
- **WHEN** 把 `ml.DecisionTreeClassifierOptions{MaxDepth: 2}` 傳給 `FitDecisionTreeClassifier`
- **THEN** 可以編譯，fit 出的樹與傳入 `ml.DecisionTreeOptions{MaxDepth: 2}` 相同，兩個別名的 doc comment 都有指名 `DecisionTreeOptions` 的 `Deprecated:` 段落
