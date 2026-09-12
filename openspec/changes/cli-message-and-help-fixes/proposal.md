# Proposal: cli-message-and-help-fixes

## Why

Three CLI findings and one performance one, all with a single right answer: #324 (CLI-15, CLI-16), #326 (CLI-18), #327 (CLI-19) and #296 (SEC-18).

- **A bad number handed back Go's own text.** `ttest single x abc` answered `strconv.ParseFloat: parsing "abc": invalid syntax`, which names neither the command nor the argument. Nine other commands did the same. Eight more already named the command and the field but still wrapped strconv's text onto the end.
- **Option keys were case-sensitive and unknown ones listed nothing.** `kmeans dt 2 NSTART 3` answered `unknown option: NSTART`, with no hint that `nstart` exists. `knn`'s `weighting` and `algorithm` values went straight into the enum, so a misspelling reached the library as an unknown mode.
- **The help table was a fixed 12 characters wide**, so `knn_neighbors` (13) pushed its description out of line. `help` also read the registry without the lock, and `read` and `env` had no Forms or Examples — `env` has nine subcommands.
- **Five commands reported a missing variable that was right there.** `clone`, `replace`, `clean`, `fillna` and `count` try a DataTable then a DataList and said "variable not found" when the variable existed but held something else, sending the caller after a typo that was not there.
- **Nine regular expressions were compiled on every call** in `lp`, `lpgen` and `datafetch`.

## What Changes

- `parseFloatArg` and `parseIntArg` report `"<cmd>: invalid <field> %q, expected a number"`. Thirteen sites use them; the eight that already named the command drop the wrapped strconv text so the whole surface reads the same way.
- `kmeans` and `knn` match option keys with `strings.ToLower`, name the command, and list what is supported. `knn` checks `weighting` and `algorithm` against the enum's own values instead of casting.
- The help table sizes its column to the longest command name. `help` and the REPL's tab completion read the registry through new locked helpers, `LookupCommand` and `SnapshotRegistry`. `read` and `env` gain Forms and Examples.
- `varTypeError` tells a missing variable from one holding the wrong type, and names the command.
- The nine regular expressions move to package-level `var`s.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `cli-failure-reporting`: an error names the command and the argument, and does not leak the standard library's text.
- `command-registry`: the registry is read through a locked accessor.

## Impact

- `cli/commands/{helpers,registry,help,read,env,clone,clean,replace,fillna,stats_dl_extra,hypothesis,timeseries,fetch,clustering,knn}.go`, `cli/repl/completer.go`, `lp/lp.go`, `lpgen/lingo.go`, `datafetch/googleMapsCommentCrawler.go`.
- User-visible: error messages change, uppercase option keys start working, and `help` lines up. Both changelogs get a CLI entry.
- `Docs/cli-dsl.md` and the CLI skill already describe `read` and `env`'s shapes; the new Forms are generated from the registry at `help` time, and `TestCommandUsageMatchesDocs` still passes.
