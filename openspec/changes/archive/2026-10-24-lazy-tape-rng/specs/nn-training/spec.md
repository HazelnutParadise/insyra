## ADDED Requirements

### Requirement: A tape seeds its generator on its first draw
The system SHALL create a tape's random generator only when something first draws from it, and SHALL seed it with the seed given to `NewTape`, or with the default seed 1 for `NewTape()` and for a zero `Tape`. Every draw SHALL come from that one stream in order, so masks and initial weights are the values an eagerly seeded `math/rand` source gives.

#### Scenario: A tape that never draws seeds nothing
- **WHEN** `NewTape(42)` returns
- **THEN** the tape SHALL hold no generator until `Dropout` or a random initialiser draws from it

#### Scenario: Draws follow the seed's stream
- **WHEN** two consecutive `Dropout` calls run on a tape made by `NewTape()`, `NewTape(0)`, `NewTape(7)`, `NewTape(-3)` or a zero `Tape`
- **THEN** each kept-or-dropped decision SHALL equal the next value of `rand.New(rand.NewSource(seed))` compared with the probability, with seed 1 for the default and the zero tape
