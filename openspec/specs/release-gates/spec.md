# release-gates Specification

## Purpose
What has to be true before `dev` is released to `main` and tagged: every change whose work ships is archived, and every CI check passes.

## Requirements

### Requirement: Shipped changes are archived before the release

Every OpenSpec change whose work is in a release SHALL be archived on `dev` before the release branch is cut. `openspec/changes/` on the release branch SHALL hold only changes whose work is not in the release.

#### Scenario: Cutting the release branch
- **WHEN** the release branch is created from `dev`
- **THEN** every change whose tasks are done is already under `openspec/changes/archive/`

### Requirement: A release needs every check green

A release SHALL NOT be merged, tagged or published while any CI check fails on the release PR or on the `main` merge commit it produces. A check that fails on every branch SHALL block the release like any other, until it is fixed or excluded by a change of its own.

#### Scenario: A check is red on the release PR
- **WHEN** any CI check on the release PR has failed
- **THEN** the release PR is not merged

#### Scenario: A check fails on every branch
- **WHEN** a check fails for a reason that is not the release's own, such as an advisory database error
- **THEN** the release still waits, and the failure is fixed or excluded in its own change first
