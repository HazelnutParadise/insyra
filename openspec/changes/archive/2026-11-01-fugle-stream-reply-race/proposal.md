## Why

When a subscribe's reply and a dropped connection arrive together, the wait picks one at random, so a command Fugle answered can report that the connection dropped. CI hit this once in `TestFugleStreamReconnectsAndResubscribes`.

## What Changes

- A reply that came in before the connection dropped answers the command.

## Impact

- `datafetch/fugle_stream.go`, both changelogs.
