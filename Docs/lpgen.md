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

### Write LP Text

```go
func (lp *LPModel) WriteLP(w io.Writer) error
```

**Description:** Writes the model to `w` in CPLEX LP format. The text is exactly what `GenerateLPFile` saves, so you can send it to a buffer, a network connection or any other writer. [`lp.Solve`](lp.md) uses it to solve a model without creating a file.

**Parameters:**

- `w`: Where to write the model. Type: `io.Writer`.

**Returns:**

- `error`: An error from `w`, or an error for an objective type other than `Minimize`/`Min`/`Minimum` or `Maximize`/`Max`/`Maximum` (any letter case). The two comment lines at the top of the text have already been written when the objective type is rejected.

```go
var buf bytes.Buffer
if err := model.WriteLP(&buf); err != nil {
    log.Fatal(err)
}
fmt.Print(buf.String())
```

### Generate LP File

```go
func (lp *LPModel) GenerateLPFile(filename string) error
```

**Description:** Saves the model to disk as a CPLEX LP file, with exactly the text `WriteLP` writes. The text goes to a temporary file in the same directory, which replaces `filename` only once everything has been written, so a failed call leaves no file behind and an existing file at `filename` keeps its content. The file is saved the way `DataTable.ToCSV` saves one: as a new file with mode 0644 that takes the old one's place, so an existing file's permissions are not kept, a symbolic link at `filename` is replaced rather than followed, and the directory has to be writable.

**Parameters:**

- `filename`: Output LP file name (e.g., "model.lp"). Type: `string`.

**Returns:**

- `error`: `nil` when the file was saved. Otherwise the reason: the file could not be created or written, or the objective type is not one `WriteLP` accepts. Nothing is logged in its place, so check it.

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
if err := lpModel.GenerateLPFile("my_model.lp"); err != nil {
    log.Fatal(err)
}
```

This example defines a simple linear programming model with two variables and constraints, and saves it as an LP file named `my_model.lp`.

## LINGO Support

The `lpgen` package also supports **LINGO**, which is a popular optimization software.

### Parse LINGO Model from Text

```go
func ParseLingo(model string) (*LPModel, error)
```

**Description:** Parses a LINGO model from text and converts it to a standard LP model. Use `LINGO > Generate > Display Model` in LINGO to get the text.

It reads these statements, each ending with `;`:

| LINGO | Becomes |
| --- | --- |
| `MIN= ...` or `MAX= ...` | the objective |
| a relation with one bare variable against numbers, such as `X >= 1` or `0 <= X <= 10` | a bound |
| any other relation, such as `3 X + Y <= 10` | a constraint |
| `@BIN(x)` | a binary variable |
| `@INT(x)` or `@GIN(x)` | a general integer variable |
| `@FREE(x)` | the bound `x free`, so `x` may be negative |
| `@BND(l, x, u)` with numbers `l` and `u` | the bound `l <= x <= u` |
| a statement starting with `!` | a comment, skipped |

Anything else is an error that gives the line the statement starts on and the statement itself, and so is a declaration whose variable cannot be read and a last statement left without its `;`. A model is never returned with part of it missing.

**Parameters:**

- `model`: LINGO model content. Type: `string`.

**Returns:**

- `*LPModel`: Parsed LP model, or `nil` when the text cannot be read.
- `error`: `nil`, or the reason: a statement it does not read, as described above, or text that cannot be read at all, such as a line of 64 KiB or more.

### Parse LINGO Model from File

```go
func ParseLingoFile(path string) (*LPModel, error)
```

**Description:** Parses a LINGO model from a file, the way `ParseLingo` parses it from text, so the same text gives the same model, or the same error, either way.

**Parameters:**

- `path`: Path to the LINGO model text file. Type: `string`.

**Returns:**

- `*LPModel`: Parsed LP model, or `nil` when the file cannot be opened or read.
- `error`: `nil`, or the reason. A missing file matches `fs.ErrNotExist` with `errors.Is`.

```go
model, err := lpgen.ParseLingoFile("model.lng")
if err != nil {
    log.Fatal(err)
}
sol, err := lp.Solve(model, lp.Options{})
```

### Deprecated LINGO names

`ParseLingoModel_str(modelStr string) *LPModel` and `ParseLingoModel_txt(filePath string) *LPModel` are the old names of `ParseLingo` and `ParseLingoFile`. They do not refuse a statement they do not read: they drop it, as they always have, including `@GIN`, `@FREE` and `@BND`, and report only text they cannot read at all, by returning `nil` and logging a warning. They are **Deprecated** and will be removed in the release after the one that deprecated them.
