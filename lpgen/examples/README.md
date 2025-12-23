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

### 4. transportation.lng
A classic transportation problem (advanced example):
- Multi-dimensional sets
- Nested @FOR and @SUM functions
- Real-world optimization scenario

**Note**: The transportation example demonstrates advanced LINGO syntax. Some features like nested @SUM within @FOR and multi-dimensional sets may require additional parsing enhancements.

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

The lpgen package currently supports the following LINGO features:

### Fully Supported
- **Comments**: Lines starting with `!`
- **Sections**: SETS, DATA, MODEL
- **@ Functions**: 
  - `@SUM(set: expression)` - Sum over a set
  - `@FOR(set: constraint)` - Generate constraints for each element
  - `@BIN(var)` - Declare binary variables
  - `@GIN(var)` - Declare general integer variables
- **Operators**: +, -, *, <=, >=, =
- **Objective functions**: MAX, MIN
- **Variable patterns**: Supports `VAR_SET` naming patterns

### Limitations
- Nested @SUM within @FOR requires each to be on separate levels
- Multi-dimensional sets (e.g., ROUTES(WAREHOUSES, CUSTOMERS)) are parsed but may need manual expansion
- Data arrays are parsed but not automatically indexed to variables in expressions

## Converting to LP Format

All example models can be converted to standard LP format using:

```bash
# Parse and convert
go run your_program.go
```

The output will be in LP format compatible with solvers like CPLEX, Gurobi, or GLPK.

## Contributing Examples

When adding new examples, please:
1. Include comments explaining the problem
2. Use meaningful variable and set names
3. Test the example to ensure it parses correctly
4. Document any advanced features used
