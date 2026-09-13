# [ lp ] Package

The `lp` package solves linear and mixed-integer programs. Give it a model built with [`lpgen`](lpgen.md) or the path of a CPLEX LP file, and it returns a `*lp.Solution` and an `error`.

By default it solves with [go-milp](https://github.com/daniel-sullivan/go-milp), a solver written in Go, so `go get` is the only setup. If you already use [GLPK](https://www.gnu.org/software/glpk/), you can select it for a call, and `lp` runs the `glpsol` program installed on your machine.

## Installation

```bash
go get github.com/HazelnutParadise/insyra/lp
```

## Quick Start

```go
model := lpgen.NewLPModel().
    SetObjective("Maximize", "3 x + 2 y").
    AddConstraint("x + y <= 4").
    AddConstraint("x + 3 y <= 6")

sol, err := lp.Solve(model, lp.Options{})
if err != nil {
    log.Fatal(err)
}
if sol.Status != lp.StatusOptimal {
    log.Fatalf("no optimal solution: %v", sol.Status)
}
fmt.Println(sol.Objective)   // 12
fmt.Println(sol.Values["x"]) // 4
fmt.Println(sol.Values["y"]) // 0
```

## Functions

### Solve

```go
func Solve(model *lpgen.LPModel, opts Options) (*Solution, error)
```

**Description:** Solves a model built with `lpgen`. The model is turned into CPLEX LP text in memory with `LPModel.WriteLP`, which is the same text `GenerateLPFile` saves, so a model solves the same way whether you pass it here or save it and call `SolveFile`.

**Parameters:**

- `model`: The model to solve. A nil model returns `ErrInvalidModel`.
- `opts`: The engine and time limit. `lp.Options{}` solves with go-milp and no time limit.

**Returns:**

- `*Solution`: The result, or nil when the error is not nil.
- `error`: Why there is no result. See [Errors](#errors).

### SolveFile

```go
func SolveFile(path string, opts Options) (*Solution, error)
```

**Description:** Solves a CPLEX LP file.

**Parameters:**

- `path`: The LP file.
- `opts`: The engine and time limit.

**Returns:**

- `*Solution`: The result, or nil when the error is not nil.
- `error`: Why there is no result. When the file cannot be read, the error wraps the operating system's error, so `errors.Is(err, fs.ErrNotExist)` reports a missing file.

## Reading a Result

A call returns a `*Solution` with a nil error whenever the solver reached an answer. "This model has no solution" is such an answer. Check the error first, then `sol.Status`:

```go
sol, err := lp.SolveFile("schedule.lp", lp.Options{TimeLimit: 30 * time.Second})
if err != nil {
    log.Fatal(err)
}
switch sol.Status {
case lp.StatusOptimal, lp.StatusFeasible:
    sol.ToDataTable().Show()
case lp.StatusStopped:
    fmt.Println("no solution was found within 30 seconds")
default:
    fmt.Println("the model is", sol.Status)
}
```

### Status

| Status | Meaning | `Objective` and `Values` |
| --- | --- | --- |
| `StatusOptimal` | The solver proved the solution optimal. | Set |
| `StatusFeasible` | The time limit stopped the search. The solution satisfies every constraint and bound, but a better one may exist. | Set |
| `StatusInfeasible` | No values satisfy every constraint and bound. | `0` and `nil` |
| `StatusUnbounded` | The objective can improve without limit. | `0` and `nil` |
| `StatusStopped` | The time limit stopped the search before any solution was found. | `0` and `nil` |

`Status.String()` returns `optimal`, `feasible`, `infeasible`, `unbounded` or `stopped`.

### Solution

```go
type Solution struct {
    Status    Status
    Objective float64
    Values    map[string]float64
    Engine    Engine
    Elapsed   time.Duration
    Nodes     int
    Log       string
}
```

- `Objective`: The objective value of the solution.
- `Values`: Each variable's value by name. Every variable in the model has an entry, including those at 0.
- `Engine`: The engine that produced the result.
- `Elapsed`: How long the call took, reading the model included.
- `Nodes`: The number of branch-and-bound nodes go-milp explored. GLPK does not report this number, so it is 0 with `EngineGLPK`.
- `Log`: With `EngineGLPK`, what `glpsol` printed, followed by its solution report. With `EngineMILP`, the warnings from reading the model, such as a missing `End` line. It is usually empty for `EngineMILP`.

### ToDataTable

```go
func (s *Solution) ToDataTable() *insyra.DataTable
```

**Description:** Returns a table with a `Variable` column and a `Value` column, one row per variable, in the order the variables first appear in the model. A solution without values gives a table with both columns, no rows, and a nil `Err()`.

```go
_ = sol.ToDataTable().ToCSV("solution.csv", false, true, false)
```

### Errors

Compare with `errors.Is`:

| Error | Returned when |
| --- | --- |
| `ErrInvalidModel` | The model cannot be solved as written. For example, the LP text has a syntax error (the message gives the line number), an `lpgen` model has an objective type other than minimize or maximize, a variable's lower bound is above its upper bound, or an integer variable has a fractional bound. |
| `ErrEngineUnavailable` | `Options.Engine` names an unknown engine, or `EngineGLPK` is selected and `glpsol` cannot be found. |
| `ErrSolverFailed` | The solver ran but produced no answer `lp` could use, for example when go-milp's simplex method fails numerically. |

## Options

```go
type Options struct {
    Engine    Engine
    TimeLimit time.Duration
}
```

- `Engine`: `lp.EngineMILP` or `lp.EngineGLPK`. The empty value selects `lp.EngineMILP`.
- `TimeLimit`: How long the search may run. Zero means no limit. When the limit is reached, the status is `StatusFeasible` with the best solution found so far, or `StatusStopped` if there is none. GLPK takes whole seconds, so `EngineGLPK` rounds the limit up: `1500 * time.Millisecond` becomes 2 seconds.

## Engines

### go-milp (default)

go-milp solves linear programs with the simplex method and integer programs with branch and bound. It is compiled into your program, with no cgo and nothing else to install.

> [!NOTE]
> When the linear relaxation at a branch-and-bound node fails numerically, go-milp v0.2.0 skips that node and can still report the result as optimal, even though the skipped part of the search might have held a better solution ([go-milp#2](https://github.com/daniel-sullivan/go-milp/issues/2)). If an integer program's answer matters, solve it once more with `EngineGLPK` and compare the objectives.

### GLPK

Select GLPK with `lp.Options{Engine: lp.EngineGLPK}`. `lp` looks for `glpsol` in two places, in this order:

1. `GLPK_PATH`, set either to the `glpsol` executable or to the directory that contains it.
2. The directories on `PATH`.

If neither has it, the call returns `ErrEngineUnavailable`.

#### Installing GLPK

| System | How |
| --- | --- |
| macOS | `brew install glpk` |
| Debian, Ubuntu | `sudo apt-get install glpk-utils` |
| Fedora | `sudo dnf install glpk-utils` |
| conda | `conda install -c conda-forge glpk` |
| Windows | Download the WinGLPK zip from [SourceForge](https://sourceforge.net/projects/winglpk/) and unzip it. It has a `w64` folder for 64-bit Windows and a `w32` folder for 32-bit Windows, each holding `glpsol.exe`. Set `GLPK_PATH` to the folder that matches your system, or add it to `PATH`. |

Run `glpsol --version` afterwards to check that the program is found. `lp`'s tests run against GLPK 5.0.

## LP File Format

`lp` reads CPLEX LP files the way GLPK 5.0 reads them, and `lpgen` writes files in this format. A file both engines accept means the same model to both, and a file GLPK rejects is rejected by go-milp's engine too, with GLPK's message and line number.

A file has these sections, in this order:

```text
Maximize
 obj: 3 x + 2 y
Subject To
 c1: x + y <= 4
 c2: x + 3 y <= 6
Bounds
 x <= 3
General
 x
End
```

`Minimize` and `Maximize` may be spelled `min`, `minimum`, `max` or `maximum`. `Subject To` may be `st` or `s.t.`, and the `Bounds`, `General`, `Integer` and `Binary` sections are optional. The rules that most often cause errors:

- A section keyword counts only at the very start of a line, so indent every other line. At the start of a line any beginning of a keyword counts too: a constraint written as `e: x >= 1` with no indent is read as `End`.
- The left side of a constraint holds only variable terms, and the right side is a single number that ends the line. `x + 3 <= 5` and a comment after the number are both errors.
- A variable with no bound has a lower bound of 0 and no upper bound. Write `x free` to let it go negative, or `-inf <= x <= 5` for an upper bound only.
- A variable under `Binary` becomes an integer, and only the sides `Bounds` left open default to 0 and 1. With `0 <= x <= 4` in `Bounds`, `x` can still be 4.
- Comments start with `\` and run to the end of the line.

GLPK reads a bound such as `x >= -foo`, where a name follows the sign, as "no lower bound". `lp` reports `missing lower bound` instead.
