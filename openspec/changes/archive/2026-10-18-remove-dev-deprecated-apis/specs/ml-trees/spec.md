## REMOVED Requirements

### Requirement: One options type for trees
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "DecisionTreeOptions is the one options type for trees".

## ADDED Requirements

### Requirement: DecisionTreeOptions is the one options type for trees

`DecisionTreeOptions` SHALL be the options type both `FitDecisionTreeClassifier` and `FitDecisionTreeRegressor` take. The aliases `DecisionTreeClassifierOptions` and `DecisionTreeRegressorOptions` SHALL NOT exist.

#### Scenario: One options type fits both trees
- **WHEN** 把 `ml.DecisionTreeOptions{MaxDepth: 2}` 分別傳給 `FitDecisionTreeClassifier` 與 `FitDecisionTreeRegressor`
- **THEN** 兩者都可以編譯並 fit 出深度不超過 2 的樹
