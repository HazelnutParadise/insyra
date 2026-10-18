## ADDED Requirements

### Requirement: Tone words are counted with negation and longest-match rules
`quant.ToneCounts` SHALL return for each text the number of positive words, negative words and total words, SHALL not count a positive word that has a negation within the negation window before it, and in substring mode SHALL match the longest term first and consume it so that a term inside a longer matched term is not counted again.

#### Scenario: A negated positive word
- **WHEN** the text is "Not good, but gain; weak loss" with a negation window of 1
- **THEN** Positive is 1, Negative is 2 and Words is 6

#### Scenario: Chinese longest match
- **WHEN** both 大漲 and 漲 are positive terms
- **THEN** 大漲 counts once and 漲 is not counted inside it

### Requirement: News is assigned to the next session once the market has closed
`quant.DailyTone` SHALL assign each article to the first trading day on or after its date, and to the next trading day when it was published at or after the close or when its date is not a trading day, and SHALL drop articles after the last trading day.

#### Scenario: Weekend and after-close news
- **WHEN** an article is published on a Saturday or on a Friday after the close
- **THEN** it counts for the following Monday

### Requirement: The aggregates do not look ahead
The z-score of the rolling net tone SHALL use only the ZWindow days before the day, never the day itself, so that changing a later day's news does not change an earlier value.

#### Scenario: A changed last day
- **WHEN** only the last day's news changes
- **THEN** every earlier z-score is unchanged

### Requirement: Unusable input is refused
Both functions SHALL return an error for a nil list or table, text that is not a string, a term in two lists or empty, a negative window, counts that are not counts, a missing column, lists of different lengths and trading days not in ascending order.

#### Scenario: A term in two lists
- **WHEN** a term is both positive and negative
- **THEN** an error names the term
