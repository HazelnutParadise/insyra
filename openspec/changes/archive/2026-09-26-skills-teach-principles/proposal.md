# Proposal: skills-teach-principles

## Why

The two agent skills, `skills/insyra` and `skills/use-insyra-cli`, try to list the library's API and every CLI command: 1,031 lines of `SKILL.md` plus five references for the Go skill, and nearly 1,900 lines plus three references for the CLI skill. That copy of the API has a cost and a flaw.

- **Cost.** `AGENTS.md` requires every feature change to update the skills as well as `Docs/`, so every API detail is written and kept in sync twice. A test and two specs pin parts of the copy: `cli-docs-registry-sync` makes two skill references mirror every command, and `quant-bootstrap` requires a bootstrap example in the skill.
- **Flaw.** A skill is installed into an agent's environment with `npx skills add HazelnutParadise/insyra/skills`, and it describes whatever version it was installed from. The project the agent works on may pin another version. An API list in the skill invites the agent to recall instead of look up, which is how an agent ends up calling a method that does not exist in the version it is compiling against. An audit found stale entries in the CLI skill already. It named a Yahoo method `quotes` that is `quote`, and it described `describe` in terms the command no longer matches.

A skill should teach what does not change from one release to the next: how to think about the library, the conventions that hold across it, how to decide between approaches, and how to find the exact API for the version in front of it. `Docs/` and the CLI's own `help` hold the details, and they ship with every version. The module cache holds `Docs/` for the exact version a project uses, and `insyra help <command>` prints the running binary's usage, forms and examples.

## What Changes

- `skills/insyra/SKILL.md` is rewritten to teach principles, thinking and where to look. It covers when to reach for Insyra, the mental model, the conventions that hold across packages (errors, concurrency, cells and numbers, column references, determinism), and a workflow for turning a question into verified code. It also gives the version-accurate lookup path and a map from kinds of question to `Docs/` pages. Its `references/` directory is removed.
- `skills/use-insyra-cli/SKILL.md` is rewritten the same way. It covers when to use one-shot commands, the REPL, `.isr` scripts or the Go DSL, the session and environment model, reproducibility, and how to discover commands from the binary. Its `references/` directory is removed.
- Nothing known is lost. Every fact in the removed content that `Docs/` did not already hold is added to `Docs/` first, found by an item-by-item audit of both skills. The same audit corrects the stale entries.
- `Docs/cli-dsl.md` becomes the only document the CLI docs sync test compares with the registry, for usage lines and for topic lists. `cli-docs-registry-sync` is modified to match.
- `quant-bootstrap`'s documentation requirement no longer asks for a skill example.
- `AGENTS.md` changes its synchronization rule. API and command details go to `Docs/` in the same change. A skill changes only when a principle, a workflow or a documentation location changes. The Agent Skills section, `cli/AGENTS.md`, `README.md` and `README_TW.md` describe the skills accordingly.
- No CHANGELOG entry: the library and the CLI do not change. Past skill edits were not in the changelog either.

## Capabilities

### New Capabilities

- `agent-skills`: what the distributed agent skills contain and where the details they point to live.

### Modified Capabilities

- `cli-docs-registry-sync`: `Docs/cli-dsl.md` is the only document compared with the registry.
- `quant-bootstrap`: the documentation requirement drops the skill example.

## Impact

- `skills/insyra/`, `skills/use-insyra-cli/`, `Docs/cli-dsl.md`, `Docs/accel.md` and the other `Docs/` pages the audit names, `cli/commands/docs_sync_test.go`, `AGENTS.md`, `cli/AGENTS.md`, `README.md`, `README_TW.md`.
- Agents with an older copy of the skills keep working. The new copy asks them to look up details they used to recall.
