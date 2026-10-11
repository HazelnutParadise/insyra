# Design: window-failures-carry-the-error

## Why not the same-length `nil` list

A list of `nil` the source's length is what a valid window produces when no position has `MinObs` observations. Returning it for an invalid option would make the two cases indistinguishable to a caller who does not check `Err()`, and it would append into a table silently. An empty list fails loudly at `AppendCols`, and with this change it also says why. Error messages keep this line's original wording.
