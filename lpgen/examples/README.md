# LINGO Example Models

This directory contains example LINGO models demonstrating various features supported by the `lpgen` package.

## Examples

### 1. simple_model.lng
A basic linear programming model demonstrating:
- Objective function (MAX)
- Linear constraints
- Non-negativity constraints
- Comments

### 2. model_with_sets.lng
Demonstrates LINGO's set-based modeling:
- SET definitions
- DATA sections
- @SUM function to sum over sets
- @FOR function to generate constraints

### 3. binary_integer_model.lng
Shows integer and binary variable declarations:
- @BIN() for binary variables
- @GIN() for general integer variables
- Mixed integer linear programming

## Usage

```go
package main

import (
    "log"
    "github.com/HazelnutParadise/insyra/lpgen"
)

func main() {
    // Parse LINGO file
    model, err := lpgen.ParseLingoFile("examples/simple_model.lng")
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate LP file
    model.GenerateLPFile("output.lp")
}
```

## LINGO Syntax Support

The lpgen package supports the following LINGO features:

- **Comments**: Lines starting with `!`
- **Sections**: SETS, DATA, MODEL
- **@ Functions**: @SUM, @FOR, @BIN, @GIN
- **Operators**: +, -, *, <=, >=, =
- **Objective functions**: MAX, MIN

## Converting to LP Format

All example models can be converted to standard LP format using:

```bash
# Parse and convert
go run your_program.go
```

The output will be in LP format compatible with solvers like CPLEX, Gurobi, or GLPK.
