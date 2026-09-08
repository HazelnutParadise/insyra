# Proposal: fix-api-review-batch-7

## Why

Six CLI defects share one shape: the command does something other than what was asked and reports success anyway. `sort dt nonexistent`, `dropcol dt nonexistent`, `droprow dt 99`, `swap dt col a b` and a CCL expression that will not compile all print their success line and exit 0, so a broken script looks like a working one. A misspelled option value — `sort … dsc`, `ttest … eqaul`, `ztest … bogus`, `clean … outliers abc` — is silently replaced by the default, which for the statistical commands means a number computed under an assumption the caller did not make. `config` writes any key and any value to disk. `accel`'s usage advertises a subcommand that does not exist and hides a flag that is parsed but not registered. `plot` and `fetch` drop arguments they do not understand. `sample` and `setcolnames` accept out-of-range counts and report success on an empty or half-cleared result. Closes #316, #317, #318, #319, #322, #323.

## What Changes

- **Failures are reported.** `sort`, `dropcol`, `droprow`, `swap`, `setcolnames` and `sample` check their target before calling the library, and every one of them (plus `ccl` and `addcolccl`) turns an error the library recorded on the table into a returned error. New `cli/commands/targets.go` holds the shared checks.
- **Option values are parsed against a closed set.** `sort`'s direction, `ttest`'s variance assumption, `ztest`'s alternative, `clean`'s standard deviation and `merge`'s trailing arguments are rejected when they are not one of the documented spellings, instead of falling back to a default.
- **`config` validates.** An unknown key is refused with the supported list; `log-level`, `no-color` and `accel-mode` accept only their own values; nothing invalid reaches the config file.
- **`accel`'s usage matches the command.** The phantom `run` subcommand is gone, `--precision` is documented and registered on the Cobra command so a one-shot invocation accepts it.
- **`plot` and `fetch` reject arguments they do not understand**, and `plot`'s usage no longer advertises options it never had.

## Capabilities

### New Capabilities

- `cli-failure-reporting`: a command that could not do what it was asked exits non-zero and says why.
- `cli-argument-validation`: an option value outside the documented set, or an argument the command does not understand, is refused rather than ignored.

### Modified Capabilities

(none)

## Impact

- `cli/commands/targets.go` (new), `sort.go`, `dropcol.go`, `droprow.go`, `swap.go`, `ccl.go`, `clean.go`, `merge.go`, `hypothesis.go`, `accel.go`, `plot.go`, `fetch.go`, `sample.go`, `setcolnames.go`, `registry.go`; `cli/env/config.go`; tests and docs.
- **BREAKING for scripts that relied on the silence**: a `.isr` script whose step used to be ignored now fails. That is the point, but it will surface typos that have been passing.
