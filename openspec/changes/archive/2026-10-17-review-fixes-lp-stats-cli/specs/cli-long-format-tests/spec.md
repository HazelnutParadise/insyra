## MODIFIED Requirements

### Requirement: anova twoway and anova repeated take a table with one row per observation

`anova twoway` SHALL take the table form, `anova twoway <table> <value> <factorA> <factorB>`, when its first argument after the mode is a DataTable variable, unless its first two arguments are both integers, which select the list form. `anova repeated` SHALL take the table form when its first argument is a DataTable variable. Otherwise both SHALL run the list form as before. The table form SHALL print the same output as the list form.

#### Scenario: Two-way ANOVA from a table
- **WHEN** a table `t` has columns `score`, `drug` and `dose`, and `anova twoway t score drug dose` is run
- **THEN** the output matches the `FactorA` and `FactorB` results of `stats.TwoWayANOVAFromTable(t, Name("score"), Name("drug"), Name("dose"))`, and of `anova twoway 2 2 …` run on the same data split into lists in first-appearance order

#### Scenario: The list form is unchanged
- **WHEN** `anova twoway 2 2 c11 c12 c21 c22` is run and the four variables are DataLists
- **THEN** the output is the same as before this change

#### Scenario: A table named like a number
- **WHEN** a DataTable is stored under the name `2` and `anova twoway 2 2 c11 c12 c21 c22` is run with four DataList variables
- **THEN** the list form runs with two levels for each factor, and the table is not used
