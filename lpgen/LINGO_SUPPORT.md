# LINGO Syntax Support in lpgen

This document provides comprehensive information about the LINGO syntax support added to the `lpgen` package.

## Overview

The `lpgen` package now supports parsing native LINGO syntax, allowing users to work with LINGO models directly without needing to generate the expanded model in LINGO first. This feature makes it easier to convert LINGO optimization models to standard LP format for use with various solvers.

## Key Features

### 1. Native LINGO Syntax Parsing
- Parse LINGO source files (`.lng` extension)
- Parse LINGO syntax from strings
- Support for LINGO sections: SETS, DATA, MODEL
- Comment handling (lines starting with `!`)

### 2. @ Function Support
The following LINGO @ functions are supported:

#### @SUM(set: expression)
Expands summation over a set of elements.

```lingo
SETS:
  PRODUCTS / P1 P2 P3 /;
ENDSETS

MODEL:
  MAX = @SUM(PRODUCTS: PROFIT * X_PRODUCTS);
END
```

Expands to:
```
MAX = PROFIT * X_P1 + PROFIT * X_P2 + PROFIT * X_P3
```

#### @FOR(set: constraint)
Generates a constraint for each element in a set.

```lingo
@FOR(PRODUCTS: X_PRODUCTS >= 0);
```

Expands to:
```
X_P1 >= 0
X_P2 >= 0
X_P3 >= 0
```

#### @BIN(variable)
Declares binary (0-1) variables.

```lingo
@BIN(X);
```

#### @GIN(variable)
Declares general integer variables.

```lingo
@GIN(Y);
```

### 3. Set and Data Definitions

#### Sets
Define collections of elements:
```lingo
SETS:
  PRODUCTS / P1 P2 P3 /;
  WAREHOUSES / W1 W2 /;
ENDSETS
```

#### Data
Define parameter values:
```lingo
DATA:
  COST = 10 20 30;
  CAPACITY = 100 80;
ENDDATA
```

## Usage Examples

### Example 1: Simple Model

```go
package main

import (
    "log"
    "github.com/HazelnutParadise/insyra/lpgen"
)

func main() {
    lingoCode := `
    MODEL:
    MAX = 3*X1 + 4*X2;
    2*X1 + 3*X2 <= 12;
    X1 >= 0;
    X2 >= 0;
    END
    `
    
    model, err := lpgen.ParseLingoSyntax(lingoCode)
    if err != nil {
        log.Fatal(err)
    }
    
    model.GenerateLPFile("output.lp")
}
```

### Example 2: Using Sets and @ Functions

```go
package main

import (
    "log"
    "github.com/HazelnutParadise/insyra/lpgen"
)

func main() {
    model, err := lpgen.ParseLingoFile("model.lng")
    if err != nil {
        log.Fatal(err)
    }
    
    // Model is now expanded and ready to use
    model.GenerateLPFile("output.lp")
    
    // Access model components
    println("Objective:", model.ObjectiveType, model.Objective)
    println("Constraints:", len(model.Constraints))
    println("Binary vars:", len(model.BinaryVars))
}
```

### Example 3: Binary and Integer Variables

```go
lingoCode := `
MODEL:
MAX = 5*X + 3*Y + 2*Z;
X + Y <= 10;
@BIN(X);
@GIN(Y);
Z >= 0;
END
`

model, _ := lpgen.ParseLingoSyntax(lingoCode)
// model.BinaryVars contains ["X"]
// model.IntegerVars contains ["Y"]
```

## API Reference

### Main Functions

#### ParseLingoSyntax
```go
func ParseLingoSyntax(source string) (*LPModel, error)
```
Parses LINGO source code and returns an LP model with @ functions expanded.

#### ParseLingoFile
```go
func ParseLingoFile(filePath string) (*LPModel, error)
```
Reads and parses a LINGO file with native syntax.

### Supporting Types

#### LingoModel
Internal representation of a LINGO model before expansion:
```go
type LingoModel struct {
    Sets        map[string][]string // Set definitions
    Data        map[string][]float64 // Data values
    Variables   map[string]string    // Variable definitions
    Objective   string               // Objective function
    Constraints []string             // Constraint expressions
}
```

## Variable Naming Conventions

The parser supports two variable naming patterns:

1. **Indexed Pattern**: `VAR_SETNAME` → expands to `VAR_elem1`, `VAR_elem2`, etc.
   ```lingo
   @SUM(PRODUCTS: X_PRODUCTS)
   → X_P1 + X_P2 + X_P3
   ```

2. **Simple Pattern**: `SETNAME` → expands to `elem1`, `elem2`, etc.
   ```lingo
   @SUM(I: 2*I)
   → 2*1 + 2*2 + 2*3
   ```

## Limitations and Future Enhancements

### Current Limitations
- Multi-dimensional sets (e.g., `ROUTES(WAREHOUSES, CUSTOMERS)`) are parsed but may need manual expansion
- Data arrays are not automatically indexed to variables in expressions
- Nested @SUM within @FOR requires careful structuring
- No support for conditional constraints (@IF)

### Planned Enhancements
- @MIN and @MAX functions for finding minimum/maximum values
- Multi-dimensional set indexing
- More complex data structure handling
- Conditional constraints with @IF
- @FREE for unbounded variables
- @SIZE to get set cardinality

## Backward Compatibility

The new LINGO syntax parser is fully backward compatible with the existing functions:
- `ParseLingoModel_txt()` - Parse generated LINGO model from file
- `ParseLingoModel_str()` - Parse generated LINGO model from string

These functions continue to work for parsing the output of `LINGO > Generate > Display Model`.

## Testing

The package includes comprehensive tests covering:
- Simple model parsing
- Set and data section parsing
- @ function expansion (@SUM, @FOR, @BIN, @GIN)
- Comment removal
- File parsing
- Backward compatibility

Run tests with:
```bash
go test ./lpgen -v
```

## Examples Directory

The `lpgen/examples/` directory contains sample LINGO models:
- `simple_model.lng` - Basic linear programming
- `model_with_sets.lng` - Using sets and @ functions
- `binary_integer_model.lng` - Binary and integer variables
- `transportation.lng` - Advanced transportation problem

## Contributing

When contributing LINGO syntax enhancements:
1. Add tests for new features
2. Update documentation
3. Ensure backward compatibility
4. Add example models demonstrating the feature

## References

- LINGO Documentation: https://www.lindo.com/
- LP Format Specification: http://lpsolve.sourceforge.net/5.5/lp-format.htm
- Insyra Project: https://insyra.hazelnut-paradise.com
