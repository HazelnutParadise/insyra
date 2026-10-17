# Proposal: thread-safety-switch-reversible

## Why

While deciding #212 on 2026-10-12, the owner kept the name `Config.Dangerously_TurnOffThreadSafety` (`core-go-names`) and asked two follow-up questions: does the name say precisely what is turned off, and can it be turned back on? Reading `atomic.go` and `config.go` answered both. The change was written on `0.4` and ported to the 0.3.x line unchanged in behaviour, since it only adds a method.

- **What it turns off.** The flag is read in exactly three places: `DataList.AtomicDo`, `DataTable.AtomicDo` and `AtomicDoAll`. When it is off, they run their callback inline without the instance's lock, which also affects any type that embeds a `*DataList` or `*DataTable`, `isr`'s included. `Config` (atomics), the global error buffer, logging and `engine/atomic.AtomicDoN` are unaffected. The doc comment says "turns off thread safety for all data structures", which is broader than the code. The owner kept the name, because "thread safety" is the consequence a user has to weigh, and asked for a precise doc comment.
- **No way back.** Only `SetDefaultConfig` sets the flag back to true, and it also resets the log level, coloured output, panic-on-error, the error hook and acceleration. A program or test that turns locking off for one phase cannot restore it without losing its other settings. The owner approved adding a way back.

## What Changes

- `Config.TurnOnThreadSafety()` turns the per-instance locking back on and changes no other setting. The name mirrors the off switch's verb, without the warning, because turning safety on is never dangerous. It logs at Info level when it actually turns locking back on, and is silent when locking is already on, the way `SetAcceleration` logs only a transition.
- `Dangerously_TurnOffThreadSafety`'s doc comment names what stops locking (`DataList.AtomicDo`, `DataTable.AtomicDo`, `AtomicDoAll`, and so every `DataList` and `DataTable` method and every type that embeds one) and what stays safe (`Config`, the error buffer, logging). It says to change the switch only while no other goroutine is using a `DataList` or `DataTable`, and names `TurnOnThreadSafety` as the way back.
- `Docs/Configuration.md` says the same.

## Capabilities

### New Capabilities

- `core-thread-safety-switch`: what the thread-safety switch turns off, and turning it back on.

### Modified Capabilities

None.

## Impact

- `config.go`; a new root test file.
- `Docs/Configuration.md`, both changelogs (Core), `AGENTS.md` (the thread-safety Key Convention line), `delivery-status.md`.
- No existing behaviour changes. `TurnOnThreadSafety` is a new method on `Config`'s unexported type, so it adds nothing a caller has to implement.
