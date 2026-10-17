## ADDED Requirements

### Requirement: An unbounded integer model is checked for an integer point

When an integer model's relaxation is unbounded, the go-milp engine SHALL look for an integer feasible point under `TimeLimit` (30 seconds when unset). It SHALL report `StatusUnbounded` only if such a point exists, `StatusInfeasible` if none exists, and `StatusStopped` if undecided. An equality over integer variables whose right side is not a multiple of the gcd of its coefficients SHALL be reported infeasible without the search.

#### Scenario: An unbounded integer model with no integer point
- **WHEN** `2 x - 2 y = 1` with `x` and `y` integers, `z` unbounded and maximised
- **THEN** the error is nil and `Status` is `StatusInfeasible`

#### Scenario: An unbounded integer model with an integer point
- **WHEN** an integer model whose relaxation is unbounded has an integer feasible point
- **THEN** the error is nil and `Status` is `StatusUnbounded`

#### Scenario: The integer check cannot decide
- **WHEN** the integer feasibility check reaches its time limit without an answer
- **THEN** the error is nil and `Status` is `StatusStopped`

### Requirement: The go-milp engine solves rows with small coefficients as written

The go-milp engine SHALL run with presolve off and SHALL scale each row whose largest coefficient is below 1, so that its largest coefficient becomes 1, before the solve. A feasible model with small coefficients SHALL NOT be reported infeasible because of the solver's absolute tolerances.

#### Scenario: Two distinct rows with coefficients near 1e-7
- **WHEN** `1e-7 x + 1e-7 y = 1` and `2e-7 x + 1e-7 y = 1.5` are solved with the objective `x + y`
- **THEN** `Status` is `StatusOptimal` and the objective is 1e7 within a relative error of 1e-6

### Requirement: TimeLimit is a best-effort bound on the search

`Options.TimeLimit` SHALL bound the solver's search and SHALL NOT be a guarantee on the total solve time. With `EngineMILP`, the limit SHALL apply from the end of the initial continuous relaxation, so the total time may exceed it.

#### Scenario: A limit on the integer search
- **WHEN** `TimeLimit` is set on an integer model with `EngineMILP`
- **THEN** the limit applies to the search that follows the first continuous relaxation, and the documentation says so

## MODIFIED Requirements

### Requirement: Solver outcomes are statuses, not errors

An infeasible model, an unbounded model, a solution found before the time limit but not proved optimal, and no solution found before the time limit SHALL be reported as `StatusInfeasible`, `StatusUnbounded`, `StatusFeasible` and `StatusStopped`, with a nil error. `Objective` and `Values` SHALL have values only for `StatusOptimal` and `StatusFeasible`, and `Values` SHALL be nil otherwise. An unbounded integer model is `StatusUnbounded` only under the integer point requirement.

#### Scenario: An infeasible model
- **WHEN** a model whose constraints contradict each other is solved
- **THEN** the error is nil, `Status` is `StatusInfeasible`, and `Values` is nil

#### Scenario: A time limit with a solution in hand
- **WHEN** an integer program reaches `TimeLimit` after finding a feasible solution
- **THEN** the error is nil, `Status` is `StatusFeasible`, and `Values` holds a value for each variable
