## ADDED Requirements

### Requirement: A LINGO statement after a comment on its line is read

`ParseLingo` and `ParseLingoFile` SHALL end a comment at the next `;` and SHALL read the statement that follows it on the same line, so a statement written after `! note;` is not dropped.

#### Scenario: A statement after an inline comment
- **WHEN** `ParseLingo("MIN=X; ! note; X >= 5;")` is called
- **THEN** the error is nil and `Bounds` contains `X >= 5`

### Requirement: A LINGO objective or relation side must be a linear expression

`ParseLingo` and `ParseLingoFile` SHALL refuse an objective or relation side that is not a linear expression made of numbers, variable names, `+` and `-`, returning a nil model and an error that names the line.

#### Scenario: An empty objective
- **WHEN** `ParseLingo("MIN= ;")` is called
- **THEN** the model is nil and the error contains `line 1`

#### Scenario: A power in an objective
- **WHEN** `ParseLingo("MIN= X**Y;")` is called
- **THEN** the model is nil and the error contains `line 1`

#### Scenario: A function call in a relation
- **WHEN** `ParseLingo("@UNKNOWN(X)=1;")` is called
- **THEN** the model is nil and the error contains `line 1`

### Requirement: WriteLP refuses an unknown objective type before writing

`(*LPModel).WriteLP` SHALL validate the objective type before it writes any byte to its writer, so a call that returns an error leaves the writer untouched.

#### Scenario: An unknown objective type
- **WHEN** a model with the objective type `upwards` is passed to `WriteLP`
- **THEN** the call returns an error and the writer receives no bytes
