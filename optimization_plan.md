# Factor Analysis Optimization Plan

## Current Issues

- PAF extraction does not converge within max iterations (50)
- Communality values do not match SPSS reference (expecting ~0.5-0.8 range, getting 0/1 pattern)
- Factor loadings may not be accurate due to convergence issues

## Required Optimizations

### 1. PAF Communality Initialization

- **Current**: Using squared diagonal elements * 0.8
- **Proposed**: Use Squared Multiple Correlations (SMC) as initial communalities
- **Rationale**: SMC provides better initial estimates for communality values in PAF

### 2. Eigenvalue Sorting

- **Current**: Assuming eigenvalues are sorted
- **Proposed**: Explicitly sort eigenvalues and eigenvectors in descending order
- **Rationale**: Ensures correct factor extraction order

### 3. Convergence Tolerance Adjustment

- **Current**: Default tolerance (likely 1e-6)
- **Proposed**: Adjust tolerance based on SPSS convergence behavior
- **Rationale**: May need looser tolerance for practical convergence

### 4. Communality Bounds

- **Current**: No bounds checking
- **Proposed**: Ensure communalities stay within [0,1] range
- **Rationale**: Prevents numerical instability

### 5. Maximum Iterations

- **Current**: 50 iterations
- **Proposed**: Increase to 100 or adjust based on convergence
- **Rationale**: Some datasets may need more iterations

### 6. Validation Against SPSS Reference

- **Current**: Partial matching for KMO/Bartlett
- **Proposed**: Full numerical validation of communalities, loadings, and scores
- **Rationale**: Ensure all outputs match SPSS exactly

## Implementation Steps

1. Modify extractPAF to use SMC for initialization
2. Add eigenvalue sorting
3. Adjust convergence parameters
4. Add bounds checking for communalities
5. Test against SPSS reference values
6. Iterate until all values match

## Expected Outcomes

- PAF converges within reasonable iterations
- Communality values match SPSS (A1: 0.718, A2: 0.573, etc.)
- Factor loadings accurate
- All diagnostic statistics correct
