# Engineering documents

Working documents for the people and agents developing Insyra. Documentation for the library's users is in `Docs/`.

| Document | What it holds | Read it when |
|---|---|---|
| [architecture.md](architecture.md) | How the system is built: components, data flow, test seams, the precision contract, the device rules, the measured thresholds and the standing assumptions | Before changing the structure, the tests, a precision or a migration |
| [decisions/](decisions/) | One file per hard-to-reverse choice, with the options that were rejected | Before proposing to change such a choice |
| [handoff.md](handoff.md) | Unfinished work that no OpenSpec change covers. Present only while there is some | Arriving to pick up work |

Two kinds of engineering document live outside this directory, because the project's OpenSpec workflow keeps them:

- **Specifications** are the capability specs in [`openspec/specs/`](/openspec/specs/).
- **Tickets** are the OpenSpec changes in [`openspec/changes/`](/openspec/changes/), one verifiable result each. Finished changes are archived in `openspec/changes/archive/`.

## How to work with them

- **Progress is where the changes are.** An open change is in `openspec/changes/`, and a finished one is archived. A change can start when nothing its `proposal.md` says it waits for is still open.
- **A change is finished by the work that delivers it**, and archived once merged.
- **When a requirement changes, the spec changes first**, through a change's spec delta, then the code.
- **A problem found and not fixed now** becomes a change, or an entry under Follow-ups in `/AGENTS.md`.
- **A decision record is not edited after it is accepted.** A later decision gets a new record, and the old one's status becomes `superseded by NNNN`.
- **Work that stops unfinished** leaves its state in the change it belongs to, or in `handoff.md` when no change covers it.
