# Design: deprecate-look-alike-window-methods

## Why not unify them into aliases

An alias has to behave exactly like the name it stands for. Making `MovingAverage` an alias of `Rolling(...).Mean()` would change its output length and its handling of gaps under an unchanged signature, so callers would get different results with nothing to warn them, and the #211 ruling keeps a deprecated name's old meaning for its last release precisely to avoid that. Two pairs cannot be unified without losing something either: `FillNaNWithMean` fills `NaN` alone, which the CLI's `fillnan` relies on, and `ExponentialSmoothing` accepts `alpha = 0`, which `EWM` refuses.

## What the deprecation says

Each doc comment names the replacement call with the method's own arguments, says how the replacement's result lines up with the old one on a fully numeric list (`Data()[w-1:]`, or `[1:]` for `Diff(1)`), names the behaviour that differs, and ends with the removal sentence the other deprecations use. The pinned differences stay in the characterization tests, which is what "keeps its old meaning" is checked by.

## The CLI

`movavg` and `diff` are the only callers outside the root package. They keep calling the deprecated methods with `// nolint:staticcheck`, as `fillnan` does, so no command's output changes in this release. When the methods are removed, each command either reproduces its current output from `Rolling`/`Diff` by dropping the leading `nil` positions, or is deprecated in favour of `rolling … mean` and `diffn`; that choice is left to the removal follow-up.
