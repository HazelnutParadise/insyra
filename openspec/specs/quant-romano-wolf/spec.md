# quant-romano-wolf Specification

## Purpose
Names which of many strategies truly beat a benchmark, with Romano and Wolf's stepdown on a jointly resampled block bootstrap, keeping the chance of any false claim at a chosen familywise level.

## Requirements

### Requirement: Romano-Wolf stepdown
`quant.RomanoWolf` SHALL take the table and options `RealityCheck` takes plus `Alpha`, and SHALL return for every column its statistic w_k = sqrt(T)·dbar_k/σ_k and `RejectedAt[k]`, the 1-based step at which column k was declared to beat the benchmark, or 0 if it was not. The test is one-sided: a column is rejected only for outperforming the benchmark.

#### Scenario: A planted strong strategy
- **WHEN** one column beats the benchmark by a wide margin among noise columns
- **THEN** that column has a non-zero `RejectedAt` and the noise columns are 0

#### Scenario: Several strategies
- **WHEN** two columns beat the benchmark by different margins
- **THEN** both have a non-zero `RejectedAt`, the stronger at the same step or an earlier one

### Requirement: Stepdown critical values
Each replication SHALL draw one sequence of time indices shared by all columns, as `RealityCheck` does, and compute w*_k = sqrt(T)·(dbar*_k − dbar_k)/σ*_k. At step s the critical value SHALL be the ceil(Alpha·B)-th largest, over the B replications, of the maximum of w* over the columns not yet rejected, and every such column with w_k above it SHALL be rejected at step s.

#### Scenario: Matches an independent implementation
- **WHEN** statistics, critical values and `RejectedAt` are computed for fixed tables and bootstrap indices
- **THEN** they equal a Python implementation of the same steps (statistics and critical values to 1e-9 relative)

#### Scenario: Everything tied
- **WHEN** every replication's maximum equals the statistics
- **THEN** no column is rejected

### Requirement: Stepdown stops
The procedure SHALL stop at the first step that rejects no column or when every column is rejected, and `CriticalValues` SHALL hold the value used at each step that ran. A stricter `Alpha` on the same table and options SHALL reject a subset of the columns a looser one rejects.

#### Scenario: A stricter level
- **WHEN** Alpha is lowered on the same table and options
- **THEN** the columns rejected are a subset of those rejected before

#### Scenario: Nothing to reject
- **WHEN** no column beats the benchmark
- **THEN** one step runs and every `RejectedAt` is 0

### Requirement: Re-estimated standard deviation
σ_k SHALL be the square root of the long-run variance ω² of d_k that `SPATest` uses and σ*_k the same estimate of the resampled series of replication b, with the autocovariances of every lag obtained by FFT.

#### Scenario: Equals the direct sum
- **WHEN** the FFT estimate is compared with `SPATest`'s direct sum for random, strongly autocorrelated, short, long and almost constant series and BlockSize 1, 2, 5, 10 and 50
- **THEN** they agree to 1e-9 relative

#### Scenario: No variance
- **WHEN** a column equals the benchmark, or a resampled series has a variance that is not finite or at most 1e-24 times its mean square
- **THEN** the call returns an error naming the column

### Requirement: Options and inputs are validated
`RomanoWolf` SHALL return an error, never a substituted default, for an `Alpha` that is NaN, at most 0 or at least 1, and for every input `RealityCheck` refuses. It SHALL accept a table with one column and read `isr` wrappers under the same lock as core types.

#### Scenario: Alpha not set
- **WHEN** `Alpha` is 0, 1 or NaN
- **THEN** the call returns an error and no result

#### Scenario: Wrapper inputs
- **WHEN** the table and benchmark are `isr` wrappers
- **THEN** the call gives the same result as for the core types
