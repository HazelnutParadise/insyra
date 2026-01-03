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

The `lpgen` package also supports **LINGO**, which is a popular optimization software.

1. **`ParseLingoModel_txt`**
   Parse LINGO model from txt file. It turns LINGO model to standard lp model.
   Go to `LINGO > Generate > Display Model` in LINGO to get the model.
   ```go
   func ParseLingoModel_txt(filePath string) *LPModel
   ```

2. **`ParseLingoModel_str`**
   Parse LINGO model from string. It turns LINGO model to standard lp model.
   Go to `LINGO > Generate > Display Model` in LINGO to get the model.
   ```go
   func ParseLingoModel_str(modelStr string) *LPModel
   ```

