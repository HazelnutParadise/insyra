# Design: cli-anova-table-forms

## Decisions

### Disambiguating table form vs list form

`anova twoway` checks if the first argument after `twoway` parses as an integer (level count). If it does, the call always routes to the list form, even if an environment table happens to be named like a number (e.g. "2"). Otherwise, if the variable resolves to a DataTable, the table form runs. `anova repeated` routes to the table form when the argument is a DataTable, and to the list form for DataLists.

### Long-format table column resolution

Table column arguments are passed as strings to the library's long-format entry points (`TwoWayANOVAFromTable`, `RepeatedMeasuresANOVAFromTable`, `FriedmanTestFromTable`), which resolve Excel-style column letters first, then exact column names.

### `friedman` command design

`friedman` supports both `friedman <table> <value> <condition> <subject>` and `friedman <s1> <s2> ...`. It prints `Q=… df=… p=…`.
