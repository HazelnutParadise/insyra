# [ lpgen ] Package

The `lpgen` package provides a simple and intuitive way to generate linear programming (LP) models and save them as LP files. It supports setting objectives, adding constraints, defining variable bounds, and specifying binary or integer variables.

## Structure of `LPModel`

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

## General Functions and Methods in `lpgen`

### Create New LP Model

```go
func NewLPModel() *LPModel
```

**Description:** Creates a new LPModel instance.

**Parameters:**

- None.

**Returns:**

- `*LPModel`: New LPModel instance. Type: `*LPModel`.

### Set Objective

```go
func (lp *LPModel) SetObjective(objType, obj string)
```

**Description:** Sets the objective function and its type (Maximize or Minimize).

**Parameters:**

- `objType`: Objective type ("Maximize" or "Minimize"). Type: `string`.
- `obj`: Objective function expression (e.g., "3 x1 + 4 x2"). Type: `string`.

**Returns:**

- None.

### Add Constraint

```go
func (lp *LPModel) AddConstraint(constr string) *LPModel
```

**Description:** Adds a constraint to the model.

**Parameters:**

- `constr`: Constraint string (e.g., "2 x1 + 3 x2 <= 12"). Type: `string`.

**Returns:**

- `*LPModel`: Updated model for chaining. Type: `*LPModel`.

### Add Bound

```go
func (lp *LPModel) AddBound(bound string) *LPModel
```

**Description:** Adds a variable bound to the model.

**Parameters:**

- `bound`: Bound string (e.g., "0 <= x1 <= 10"). Type: `string`.

**Returns:**

- `*LPModel`: Updated model for chaining. Type: `*LPModel`.

### Add Binary Variable

```go
func (lp *LPModel) AddBinaryVar(varName string) *LPModel
```

**Description:** Adds a binary variable to the model.

**Parameters:**

- `varName`: Name of the binary variable (e.g., "x3"). Type: `string`.

**Returns:**

- `*LPModel`: Updated model for chaining. Type: `*LPModel`.

### Add Integer Variable

```go
func (lp *LPModel) AddIntegerVar(varName string) *LPModel
```

**Description:** Adds an integer variable to the model.

**Parameters:**

- `varName`: Name of the integer variable (e.g., "x4"). Type: `string`.

**Returns:**

- `*LPModel`: Updated model for chaining. Type: `*LPModel`.

### Generate LP File

```go
func (lp *LPModel) GenerateLPFile(filename string)
```

**Description:** Generates an LP file based on the current model and saves it to disk.

**Parameters:**

- `filename`: Output LP file name (e.g., "model.lp"). Type: `string`.

**Returns:**

- None.

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

## LINGO Support

The `lpgen` package also supports **LINGO**, which is a popular optimization software.

### Parse LINGO Model from Text File

```go
func ParseLingoModel_txt(filePath string) *LPModel
```

**Description:** Parses a LINGO model from a text file and converts it to a standard LP model. Use `LINGO > Generate > Display Model` to export the model.

**Parameters:**

- `filePath`: Path to the LINGO model text file. Type: `string`.

**Returns:**

- `*LPModel`: Parsed LP model. Type: `*LPModel`.

### Parse LINGO Model from String

```go
func ParseLingoModel_str(modelStr string) *LPModel
```

**Description:** Parses a LINGO model from a string and converts it to a standard LP model. Use `LINGO > Generate > Display Model` to export the model.

**Parameters:**

- `modelStr`: LINGO model content as a string. Type: `string`.

**Returns:**

- `*LPModel`: Parsed LP model. Type: `*LPModel`.
