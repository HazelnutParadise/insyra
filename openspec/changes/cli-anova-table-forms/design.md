# Design: cli-anova-table-forms

## Decisions

### The variable's type picks the form, not a new mode name

`anova twoway` and `anova repeated` keep their names. When the first argument after the mode is a DataTable variable, the command runs the table form; otherwise it runs the list form exactly as before. The alternatives were a separate mode (`anova twoway-table`) or a keyword (`anova twoway from t …`). Both add a spelling to learn for the same test, and the variable's type is already known, so nothing is guessed. `describe` already behaves differently for a list and a table. The existing list form's first argument is an integer level count for `twoway` and a DataList for `repeated`, neither of which is a DataTable variable, so no existing script changes meaning.

### The table forms call the library's table entry points

The CLI resolves each column token with the one-token rule, turns it into a library selector with `colSelector`, and calls `TwoWayANOVAFromTable`, `RepeatedMeasuresANOVAFromTable` or `FriedmanTestFromTable`. Levels, refusals and results are therefore the library's, and the CLI adds nothing a Go user would not get.

### `friedman` is its own command

The CLI names its test commands after the test (`ttest`, `ztest`, `chisq`, `anova`), and the Friedman test is not an ANOVA, so it does not go under `anova`. It gets the list form too, mirroring `anova repeated`, because a command that only reads tables would be the one test command with no list form. The other rank tests stay out of this change: nobody asked for them, and each needs its own form design.

### Output

The table forms print the same line as the list forms (`FA=… pA=… FB=… pB=…` for two-way, `F=… p=…` for repeated measures), so a script can switch forms without changing what it parses. `friedman` prints `Q=… df=… p=…`.
