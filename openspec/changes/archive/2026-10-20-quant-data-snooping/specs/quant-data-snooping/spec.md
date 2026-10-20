## ADDED Requirements

### Requirement: Relative performance against a benchmark
`quant.RealityCheck` and `quant.SPATest` SHALL take a T×N performance table whose column k is strategy k and whose row t is period t, higher being better, and SHALL compute d_k,t as column k minus `DataSnoopingOptions.Benchmark` at row t, or minus 0 when the benchmark is nil. The null hypothesis is that no strategy has a positive expected d.

#### Scenario: No benchmark
- **WHEN** `Benchmark` is nil
- **THEN** each column is compared with 0, as for a table of excess returns

#### Scenario: Benchmark of the wrong length
- **WHEN** the benchmark does not have T values
- **THEN** both functions return an error

### Requirement: Joint block resampling
Each bootstrap replication SHALL draw one sequence of T time indices, by the stationary bootstrap (the default) or the moving block bootstrap with `BlockSize`, and SHALL apply that sequence to every column, so that the correlation between strategies is kept. The indices SHALL come from the same seeded stream `BlockBootstrap` uses, so the same table and options give bit-identical results.

#### Scenario: Same seed, same result
- **WHEN** either function is called twice with the same table and options
- **THEN** the two results are identical

#### Scenario: Indices match BlockBootstrap
- **WHEN** a replication's indices are drawn with a seed and block settings
- **THEN** they are the positions `BlockBootstrap` with the same seed, scheme and block size, a horizon of T and one path per replication would draw

### Requirement: BlockBootstrap output is unchanged
Splitting `BlockBootstrap`'s resampling into drawing indices and reading values SHALL NOT change any value `BlockBootstrap` returns for a given input and configuration.

#### Scenario: Golden paths
- **WHEN** `BlockBootstrap` is called with the seeds its existing tests pin
- **THEN** it returns the same paths as before

### Requirement: White's Reality Check
`RealityCheck` SHALL return the statistic V = max(max_k sqrt(T)·dbar_k, 0), the column with the largest dbar_k, and the p-value equal to the share of replications whose V* = max(0, max_k sqrt(T)·(dbar*_k − dbar_k)) is strictly greater than V.

#### Scenario: A strong strategy
- **WHEN** one column beats the benchmark by a wide margin in every period
- **THEN** the p-value is near 0 and that column is reported as the best

#### Scenario: No strategy beats the benchmark
- **WHEN** every column is noise around the benchmark
- **THEN** the p-value is not small

### Requirement: Hansen's SPA test
`SPATest` SHALL return T_SPA = max(max_k sqrt(T)·dbar_k/ω_k, 0) and three p-values, each the share of replications with T* strictly greater than T_SPA, where T* = max(0, max_k sqrt(T)·(dbar*_k − g(dbar_k))/ω_k) and ω_k is the same for the statistic and every replication. The lower p-value SHALL use g(x) = max(0, x), the upper g(x) = x and the consistent g(x) = x when x ≥ −sqrt(ω_k²/T·2·log log T) and 0 otherwise.

#### Scenario: Ordering of the three p-values
- **WHEN** any table is tested
- **THEN** the lower p-value is at most the consistent one, which is at most the upper one

#### Scenario: A poor strategy does not hide a good one
- **WHEN** a strong column is tested together with many columns far below the benchmark
- **THEN** the consistent p-value stays near 0

### Requirement: Long-run variance of the SPA test
ω_k² SHALL be γ_0 + 2·Σ_{i=1}^{T−1} κ(T, i)·γ_i, where γ_i is the lag-i autocovariance of d_k divided by T and κ(T, i) = ((T−i)/T)·(1−q)^i + (i/T)·(1−q)^(T−i) with q = 1/BlockSize, as in Hansen (2005). A column whose ω_k² is not positive SHALL be an error.

#### Scenario: Matches an independent implementation
- **WHEN** ω_k, dbar_k and T_SPA are computed for a fixed table
- **THEN** they equal a Python implementation of the paper's formulas to 1e-12

#### Scenario: A column equal to the benchmark
- **WHEN** a column equals the benchmark in every period
- **THEN** `SPATest` returns an error naming that column

### Requirement: Options are validated
Both functions SHALL return an error, never a substituted default, for a table with fewer than 3 rows or no columns, a cell that is not a finite number, `Replications` below 1, `BlockSize` below 1 or above T, and a `ResamplingScheme` that is not one of its constants. `ResamplingScheme` SHALL have `String()`.

#### Scenario: Bad block size
- **WHEN** `BlockSize` is 0 or greater than T
- **THEN** the call returns an error and no result

#### Scenario: Non-numeric cell
- **WHEN** a cell of the table is text or NaN
- **THEN** the error names its column and row index
