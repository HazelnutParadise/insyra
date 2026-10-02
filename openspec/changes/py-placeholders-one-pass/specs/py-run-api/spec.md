## ADDED Requirements

### Requirement: Placeholders are replaced in one pass with Python literals

`RunCodef`, `RunFilef`, `RunCodefContext`, `RunFilefContext` and `Run` SHALL find the placeholders `$v1`, `$v2`, … in one pass over the template, each read by its whole number, and SHALL replace each with the Python literal of the argument at that position. Text inserted for one placeholder SHALL never be searched for another. A placeholder whose number is past the last argument, or written with a leading zero, SHALL be left as written. Only the arguments the template uses SHALL be converted. An argument that cannot be written as a Python literal SHALL make the call return an error naming its placeholder before Python starts, and SHALL never be written into the script as formatted text.

#### Scenario: A value holding a placeholder
- **WHEN** the template is `title = $v1` and `label = $v2` on two lines, the first argument is the text `$v2` and the second is `+__import__('os').system('id')+`
- **THEN** the script holds `title = "$v2"` and `label = "+__import__('os').system('id')+"`

#### Scenario: Ten or more arguments
- **WHEN** the template is `x = $v10 + $v1` and the arguments are the texts `a` to `j`
- **THEN** the script holds `x = "j" + "a"`

#### Scenario: A value Python cannot read
- **WHEN** the template uses `$v1` and the argument is `[]any{"__import__('os').system('id'),", math.NaN()}`
- **THEN** the call returns an error naming `$v1` and Python does not start

#### Scenario: A non-finite float in a slice
- **WHEN** the template uses `$v1` and the argument is `[]float64{1, math.NaN()}`, or holds an infinity
- **THEN** the call returns an error naming `$v1`, as for a single `math.NaN()`

#### Scenario: An argument the template does not use
- **WHEN** the template is `x = $v2` and the arguments are `math.NaN()` and `3`
- **THEN** the script holds `x = 3` and the call does not fail
