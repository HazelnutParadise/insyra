## Why

`ParseLingoModel_str` and `ParseLingoModel_txt`, deprecated for one release, are removed in 0.4. `remove-dev-deprecated-apis` missed the `lpgen-model-files` spec because its search pattern did not match these names.

## What Changes

- **BREAKING** The requirement that kept the two functions alive is removed.
- The requirement that compares `ParseLingo` with the old parser no longer cites it.
- The refusal requirement is restated for `ParseLingo` and `ParseLingoFile` alone.

## Impact

- Spec: `lpgen-model-files`. Code, docs and both changelogs were updated with `remove-dev-deprecated-apis`.
