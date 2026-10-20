# ta-oscillators Specification

## Purpose
Oscillator and momentum indicators (CCI, momentum, rate of change, ultimate oscillator, stochastic RSI, AR/BR, CR, force index, Elder-ray) with TA-Lib's or the standard definitions.

## Requirements

### Requirement: Oscillators follow their definitions
`ta.CCI`, `ta.Momentum`, `ta.ROC`, `ta.UltimateOscillator` and `ta.StochRSI` SHALL equal TA-Lib's CCI, MOM, ROC, ULTOSC and STOCHRSI, `ta.ROC` SHALL return a percentage, and `ta.StochRSI` SHALL smooth with the same KDOptions as `ta.KD`.

#### Scenario: A percentage
- **WHEN** the price rises from 100 to 105 over one row
- **THEN** ROC is 5

### Requirement: Taiwanese sentiment indicators are available
`ta.ARBR` SHALL return AR as 100 times the sum of high minus open over the sum of open minus low and BR as 100 times the sum of the rise of the high above the previous close over the fall of the low below it, over n rows, and `ta.CR` SHALL compare each day with the previous row's midpoint, chosen by CROptions.Midpoint from (high + low) / 2, (2 close + high + low) / 4, (high + low + close) / 3 and (open + high + low + close) / 4.

#### Scenario: The open is needed for one midpoint
- **WHEN** the midpoint is CRMidOHLC and no open is given
- **THEN** an error names CROptions.Open

### Requirement: Elder's indicators are available
`ta.ForceIndex` SHALL return the n-row EMA of the price change times the volume, and `ta.ElderRay` the high and the low minus the n-row EMA of the close.

#### Scenario: Force with n 1
- **WHEN** n is 1
- **THEN** the result is the raw change times volume from row 1

### Requirement: Unusable input is refused
Each oscillator function SHALL return an error naming the row index for a missing, non-numeric or non-finite value, and an error for a period below 1, windows that are not in increasing order, a high below its low, an unknown midpoint or method, and series of different lengths.

#### Scenario: Windows out of order
- **WHEN** the short window is 14 and the medium is 7
- **THEN** an error is returned
