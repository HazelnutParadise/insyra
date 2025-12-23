# [ lpgen ] Package

The `lpgen` package provides a simple and intuitive way to generate linear programming (LP) models and save them as LP files. It supports setting objectives, adding constraints, defining variable bounds, and specifying binary or integer variables.

#### Structure of `LPModel`

```go
type LPModel struct {
	Objective     string   // Objective function (e.g., "3 x1 + 4 x2")
	ObjectiveType string   // Type of objective (e.g., "Maximize" or "Minimize")
	Constraints   []string // List of constraints (e.g., "2 x1 + 3 x2 <= 12")
	Bounds        []string // List of variable bounds (e.g., "0 <= x1 <= 10")
	BinaryVars    []string // List of binary variables (e.g., "x3")
	IntegerVars   []string // List of integer variables (e.g., "x4")
}
```

The `LPModel` struct is the core of the `lpgen` package, allowing users to define the essential elements of a linear programming model:

- **Objective**: Defines the function to maximize or minimize.
- **Constraints**: A list of constraints that must be satisfied.
- **Bounds**: Defines the bounds for the variables.
- **BinaryVars**: Specifies which variables are binary.
- **IntegerVars**: Specifies which variables are integers.

#### General Functions and Methods in `lpgen`

1. **`NewLPModel`**:  
   Creates a new LPModel instance.
   ```go
   func NewLPModel() *LPModel
   ```

2. **`SetObjective`**:  
   Sets the objective function and its type (Maximize or Minimize).
   ```go
   func (lp *LPModel) SetObjective(objType, obj string)
   ```
   - `objType`: Either "Maximize" or "Minimize".
   - `obj`: The objective function (e.g., "3 x1 + 4 x2").

3. **`AddConstraint`**:  
   Adds a constraint to the model.
   ```go
   func (lp *LPModel) AddConstraint(constr string) *LPModel
   ```
   - `constr`: A string representing the constraint (e.g., "2 x1 + 3 x2 <= 12").

4. **`AddBound`**:  
   Adds a variable bound to the model.
   ```go
   func (lp *LPModel) AddBound(bound string) *LPModel
   ```
   - `bound`: A string representing the bound (e.g., "0 <= x1 <= 10").

5. **`AddBinaryVar`**:  
   Adds a binary variable to the model.
   ```go
   func (lp *LPModel) AddBinaryVar(varName string) *LPModel
   ```
   - `varName`: The name of the binary variable (e.g., "x3").

6. **`AddIntegerVar`**:  
   Adds an integer variable to the model.
   ```go
   func (lp *LPModel) AddIntegerVar(varName string) *LPModel
   ```
   - `varName`: The name of the integer variable (e.g., "x4").

7. **`GenerateLPFile`**:  
   Generates an LP file based on the current model and saves it to disk.
   ```go
   func (lp *LPModel) GenerateLPFile(filename string)
   ```
   - `filename`: The name of the file to be created (e.g., "model.lp").
   
   The function writes the model data to the LP file in the following format:
   - Objective type (Maximize or Minimize)
   - Objective function
   - Constraints
   - Bounds
   - General (Integer variables)
   - Binary (Binary variables)
   - End

#### Example Usage

```go
lpModel := lpgen.NewLPModel()

// Set objective function to maximize
lpModel.SetObjective("Maximize", "3 x1 + 4 x2")

// Add constraints
lpModel.AddConstraint("2 x1 + 3 x2 <= 20")
lpModel.AddConstraint("4 x1 + 2 x2 <= 30")

// Add bounds for variables
lpModel.AddBound("0 <= x1 <= 10")
lpModel.AddBound("0 <= x2 <= 10")

// Add integer and binary variables
lpModel.AddIntegerVar("x1")
lpModel.AddBinaryVar("x2")

// Generate LP file
lpModel.GenerateLPFile("my_model.lp")
```

This example defines a simple linear programming model with two variables and constraints, and saves it as an LP file named `my_model.lp`.

#### LINGO Support

The `lpgen` package supports **LINGO**, a popular optimization software, in two ways:

##### 1. Parse Generated LINGO Models

These functions parse the model output from LINGO (via `LINGO > Generate > Display Model`):

**`ParseLingoModel_txt`**
   Parse LINGO model from txt file. It turns LINGO model to standard lp model.
   Go to `LINGO > Generate > Display Model` in LINGO to get the model.
   ```go
   func ParseLingoModel_txt(filePath string) *LPModel
   ```

**`ParseLingoModel_str`**
   Parse LINGO model from string. It turns LINGO model to standard lp model.
   Go to `LINGO > Generate > Display Model` in LINGO to get the model.
   ```go
   func ParseLingoModel_str(modelStr string) *LPModel
   ```

##### 2. Parse Native LINGO Syntax

These functions parse LINGO source code directly, including support for LINGO @ functions:

**`ParseLingoSyntax`**
   Parse LINGO source code with @ functions like @SUM, @FOR, @BIN, @GIN.
   ```go
   func ParseLingoSyntax(source string) (*LPModel, error)
   ```

**`ParseLingoFile`**
   Read and parse a LINGO file with native syntax.
   ```go
   func ParseLingoFile(filePath string) (*LPModel, error)
   ```

##### Supported LINGO Features

- **@ Functions**:
  - `@SUM(set: expression)` - Expands to sum of terms
  - `@FOR(set: constraint)` - Generates multiple constraints
  - `@BIN(var)` - Declares binary variables
  - `@GIN(var)` - Declares general integer variables

- **Sections**:
  - `SETS:` ... `ENDSETS` - Define sets of elements
  - `DATA:` ... `ENDDATA` - Define data values
  - `MODEL:` ... `END` - Define the optimization model

- **Comments**: Lines starting with `!` are treated as comments

##### Example: Simple LINGO Model

```lingo
! Simple Production Planning Model
MODEL:
MAX = 3*X1 + 4*X2;

! Resource constraints
2*X1 + 3*X2 <= 12;
4*X1 + 2*X2 <= 16;

! Non-negativity constraints
X1 >= 0;
X2 >= 0;
END
```

```go
model, err := lpgen.ParseLingoFile("model.lng")
if err != nil {
    log.Fatal(err)
}
model.GenerateLPFile("output.lp")
```

##### Example: LINGO with Sets and @ Functions

```lingo
! Production Planning with Sets
SETS:
PRODUCTS / P1 P2 P3 /;
ENDSETS

DATA:
PROFIT = 10 15 20;
ENDDATA

MODEL:
MAX = @SUM(PRODUCTS: PROFIT * X);
@FOR(PRODUCTS: X >= 0);
END
```

```go
model, err := lpgen.ParseLingoSyntax(lingoSource)
if err != nil {
    log.Fatal(err)
}
model.GenerateLPFile("output.lp")
```

##### Example: Binary and Integer Variables

```lingo
MODEL:
MAX = 5*X + 3*Y + 2*Z;
X + Y <= 10;

@BIN(X);
@GIN(Y);

Z >= 0;
END
```

```go
model, err := lpgen.ParseLingoSyntax(lingoSource)
if err != nil {
    log.Fatal(err)
}
// model.BinaryVars will contain ["X"]
// model.IntegerVars will contain ["Y"]
model.GenerateLPFile("output.lp")
```
