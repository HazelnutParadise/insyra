# Design: deprecate-look-alike-window-methods

## Why not unify them into aliases

An alias has to behave exactly like the name it stands for. Making `MovingAverage` an alias of `Rolling(...).Mean()` would change its output length and its handling of gaps under an unchanged signature, so callers would get different results with nothing to warn them, and the #211 ruling keeps a deprecated name's old meaning until removed in 0.4 precisely to avoid that. Two pairs cannot be unified without losing something either: `FillNaNWithMean` fills `NaN` alone, which the CLI's `fillnan` relies on, and `ExponentialSmoothing` accepts `alpha = 0`, which `EWM` refuses.

## What the deprecation says

Each doc comment names the replacement call with the method's own arguments, says how the replacement's result lines up with the old one on a fully numeric list, names the behaviour that differs, and notes removal in 0.4. The pinned differences stay in the characterization tests, which is what "keeps its old meaning" is checked by.
