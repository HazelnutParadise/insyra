# Proposal: cli-docs-match-registry

## Why

The CLI is documented in three hand-written places: the command index in `Docs/cli-dsl.md`, and the two agent-skill references `cli-command-usage.md` and `cli-command-guide.md`. Two more pages list the commands by topic: the command groups in `Docs/cli-dsl.md` and the skill's `cli-commands.md`. Nothing compares any of them with the commands themselves, and they have drifted.

Checked against all 117 registered commands: `accel` is missing from all three usage documents and from the command groups. `read` is documented without `ragged` and `trimspace`, which it accepts. `plot` still advertises `[options...]`, which batch 7 removed. `save … sql` documents `[rownames]` where the command takes `[rownames [true|false]]`. The usage reference says a JSON load only warns about `headers`, but `load` returns an error. The guide's examples for `merge`, `count`, `ttest`, `ztest`, `anova`, `ftest` and `chisq` do not run as written: `merge x1 x2 inner strict` fails with `invalid merge direction: inner`, `count x` fails because the value is required, and the statistics examples stop before their arguments.

Three commands' own `Usage` strings are wrong too, which is the text `help` prints: `pca` and `regression` accept `as <var>` without saying so, and `count` marks as optional a value it requires.

Closes #321.

## What Changes

- **A test compares the usage documents with the registry.** For every registered command, each of the three documents must have an entry, and the entry's usage line must equal the command's `Usage` exactly. A document entry for something that is not a command fails too, apart from Cobra's own `completion` and the guide's `load sql` / `save sql` sub-sections. The test refuses to pass if it parses implausibly few entries.
- **A second test checks the two topic lists.** The command groups and `cli-commands.md` must name every registered command. The groups gain `accel`, `describe` and the deprecated `fillnan`.
- **Each document's usage line mirrors `Usage`.** Where a document gave more detail than `Usage` (the forms of `fetch`, `load` and `rolling`), the detail moves to a separate line, which is how the registry already keeps it (`Forms`). The usage reference gains the Parquet options the old `load` line carried.
- **`accel` is documented** in all three usage documents.
- **The broken examples are replaced by complete invocations**, each run in an isolated home directory before being written down. Beyond the seven the review named, twelve more in the guide were templates rather than examples: `insyra clean x nan|nil|strings|outliers` is read by a shell as a pipeline, and `insyra set x 0 0 <value>` also fails because `set` takes a column letter. `percentile x 0.9` asked for the 0.9th percentile because `<p>` runs from 0 to 100; the example is now `90`, with a note on the scale.
- **`pca`, `regression` and `count` get correct `Usage` strings**, and their usage errors match. The usage error for `save … sql` shows `[rownames [true|false]]`.
- The JSON note and the `save … sql` `rownames` note are corrected, and the four pages stop claiming to be generated from `help`.
- The unreleased changelog entry from batch 7 said `accel --precision` "works" in one-shot mode. The flag is accepted, but nothing reads it; the entry now says so.

## Capabilities

### New Capabilities

- `cli-docs-registry-sync`: the CLI documentation describes the commands that exist, with the options they accept.

### Modified Capabilities

(none)

## Impact

- `cli/commands`: `pca.go`, `regression.go`, `stats_dl_extra.go` (`count`), `db_save.go` (usage error); `docs_sync_test.go` and `usage_accuracy_test.go`.
- `Docs/cli-dsl.md`, `skills/use-insyra-cli/references/cli-command-usage.md`, `cli-command-guide.md` and `cli-commands.md`; `CHANGELOG.md` and `CHANGELOG_TW.md`; `api-review.md`; `AGENTS.md`; `delivery-status.md`.
- Item 6 of the issue, `insyra --no-color show x` not running, was fixed by batch 4 and was confirmed to run with no colour codes; nothing changes for it.
- The tests check usage lines and command names, not examples: running every example needs data prepared for each one. The examples changed here were run by hand.
- Running the examples turned up two behaviour problems. They are recorded as `AGENTS.md` follow-ups rather than fixed here, because each changes what a command does. `count`, `find` and `replace` never match an integer: the literal parser returns `int`, while CSV loads and restored variables hold `int64`, so `count x 3` prints 0 on data containing 3. Several commands accept arguments they ignore, `accel --precision` among them. The guide's examples use string values until the first is fixed.
