## Purpose

The agent skills distributed under `skills/` teach agents how to use Insyra: when to reach for it, how to think about it, the conventions that hold across it, and where to find the exact API. `Docs/` and the CLI's own `help` hold the API and command details, and they ship with every version.

## ADDED Requirements

### Requirement: A skill teaches principles and where to look, not a catalogue

`skills/insyra/SKILL.md` and `skills/use-insyra-cli/SKILL.md` SHALL teach when to use the library or the CLI, its mental model, the conventions that hold across it, how to choose between approaches and verify a result, and where to find documentation. They SHALL NOT list the library's API or the CLI's command set. The skills SHALL NOT carry reference files that repeat `Docs/`.

#### Scenario: An agent needs a function's signature
- **WHEN** an agent using `skills/insyra` needs the exact name and parameters of a function
- **THEN** the skill tells it how to look them up for the version in use, and does not offer a list of its own to recall from

#### Scenario: A function is added to the library
- **WHEN** a change adds or alters a function or a CLI command
- **THEN** `Docs/` is updated in the same change, and the skills need no edit unless a principle, a workflow or a documentation location changed

### Requirement: Lookups start from the version in use

Each skill SHALL direct an agent to documentation for the version the project or binary actually uses, before any other source. For the library, that is the module directory given by `go list -m -f '{{.Dir}}' github.com/HazelnutParadise/insyra`, with its `Docs/`, `go doc`, and the source and tests for that version. For the CLI, it is `insyra help` and `insyra help <command>`. Released documentation at the matching tag, the documentation site and pkg.go.dev SHALL come after.

#### Scenario: A project pins an older release
- **WHEN** the project's `go.mod` requires a release older than the skill
- **THEN** the skill's first lookup lands on that release's `Docs/` and symbols, not on the newest documentation

### Requirement: Nothing is lost when a skill drops a detail

Before content leaves a skill, every fact in it that `Docs/` (or, for the CLI, `help`) does not already hold SHALL be added there. A statement found to contradict the code SHALL be corrected, not moved.

#### Scenario: The CLI skill's command guide is removed
- **WHEN** `skills/use-insyra-cli/references/` is deleted
- **THEN** every behaviour it described that `Docs/cli-dsl.md` and `help` lacked is already in `Docs/cli-dsl.md`
