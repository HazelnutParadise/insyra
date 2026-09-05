package quant

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
)

// PortfolioObjective selects what OptimizePortfolio maximises or minimises.
type PortfolioObjective uint8

const (
	// MinimumVariance minimises w'Sigma w. It is the zero value, so an empty
	// PortfolioConfig asks for the global minimum-variance portfolio under the
	// default long-only bounds.
	MinimumVariance PortfolioObjective = iota
	// TargetReturn minimises w'Sigma w subject to mu'w equal to
	// PortfolioConfig.TargetReturn.
	TargetReturn
	// MaximumSharpe maximises (mu'w - RiskFreeRate) / sqrt(w'Sigma w) by
	// searching along the efficient frontier.
	MaximumSharpe
)

// PortfolioConfig configures OptimizePortfolio, OptimizePortfolioMoments, and
// EfficientFrontier. Only Tolerance and MaxIterations are defaulted; every
// other field is used exactly as given, and an inconsistent one is an error
// rather than a silent correction.
type PortfolioConfig struct {
	// Objective selects the problem being solved. The zero value is
	// MinimumVariance.
	Objective PortfolioObjective
	// TargetReturn is the per-period expected return the portfolio must hit,
	// used only when Objective is TargetReturn. It must lie inside the range
	// attainable under the bounds.
	TargetReturn float64
	// RiskFreeRate is the per-period risk-free rate. It is used by
	// MaximumSharpe and to report SharpeRatio; it never affects
	// MinimumVariance or TargetReturn weights.
	RiskFreeRate float64
	// MinWeight is the per-asset lower bound in table-column order. nil means
	// 0 for every asset. Pass a negative value to allow a short position.
	MinWeight []float64
	// MaxWeight is the per-asset upper bound in table-column order. nil means
	// 1 for every asset.
	MaxWeight []float64
	// Tolerance is the stopping threshold on the projected-gradient step,
	// measured in weight units. Defaults to 1e-10 when not positive.
	Tolerance float64
	// MaxIterations caps the solver. Defaults to 10000 when not positive.
	// Hitting it reports Converged false with the best weights found, never an
	// error.
	MaxIterations int
}

// PortfolioResult holds one solved portfolio. Weights and AssetNames use the
// input column order.
type PortfolioResult struct {
	// Weights holds the optimal weight per asset. They sum to 1 and lie within
	// the configured bounds.
	Weights []float64
	// AssetNames holds the asset names in the same order as Weights.
	AssetNames []string
	// ExpectedReturn is mu'w, a per-period expected return.
	ExpectedReturn float64
	// Variance is w'Sigma w, a per-period variance.
	Variance float64
	// Volatility is sqrt(Variance), a per-period standard deviation.
	Volatility float64
	// SharpeRatio is (ExpectedReturn - RiskFreeRate) / Volatility, per period
	// like its inputs. Multiply by sqrt(periods per year) to annualize it.
	SharpeRatio float64
	// Iterations is the number of projected-gradient steps taken across every
	// inner solve.
	Iterations int
	// Converged reports whether every solve reached Tolerance before
	// MaxIterations. False means Weights are the best found so far.
	Converged bool
}

// Weight returns the weight of the named asset. It returns false when the name
// is not among AssetNames, and on a nil receiver.
func (r *PortfolioResult) Weight(name string) (float64, bool) {
	if r == nil {
		return 0, false
	}
	for i, assetName := range r.AssetNames {
		if assetName == name {
			return r.Weights[i], true
		}
	}
	return 0, false
}

// OptimizePortfolio solves a mean-variance problem from a table of aligned
// per-period returns: one column per asset, one row per period. Expected
// returns are the column means and Sigma is the sample covariance with the
// n-1 denominator.
//
// Weights always satisfy sum(w) = 1 and cfg.MinWeight[i] <= w[i] <=
// cfg.MaxWeight[i], which default to the long-only box [0, 1]. Short positions
// therefore require an explicit negative MinWeight.
//
// The problem is solved by accelerated projected gradient with an exact
// Euclidean projection onto the bounded simplex; TargetReturn adds the return
// equality through an augmented Lagrangian and MaximumSharpe searches along
// the frontier. Reaching MaxIterations is reported as Converged false with the
// best weights found, not as an error.
//
// Returns an error if returns is nil or has fewer than 2 columns, if there are
// fewer observations than assets plus one, if the columns have different
// lengths, if a cell is unreadable or non-finite (the error names the column
// and the one-based row), if the bounds are the wrong length, inverted, or
// infeasible, if a TargetReturn is outside the attainable range, or if
// Objective is unknown.
func OptimizePortfolio(returns insyra.IDataTable, cfg PortfolioConfig) (*PortfolioResult, error) {
	mean, cov, names, err := portfolioMoments(returns)
	if err != nil {
		return nil, err
	}
	// The covariance is a sample covariance and so positive semidefinite by
	// construction; only caller-supplied moments need the check.
	return optimize("OptimizePortfolio", mean, cov, names, cfg)
}

// OptimizePortfolioMoments solves the same problem from moments the caller
// supplies, which is the seam for shrunk, forecast, or otherwise estimated
// inputs. mean is the per-period expected return per asset and cov is its
// covariance matrix in the same order; names may be nil, in which case assets
// are named "1", "2", and so on.
//
// In addition to the errors OptimizePortfolio reports, cov must be square,
// agree with mean in length, be symmetric, be positive semidefinite, and hold
// only finite values, and names (when given) must have one entry per asset.
func OptimizePortfolioMoments(mean []float64, cov [][]float64, names []string, cfg PortfolioConfig) (*PortfolioResult, error) {
	const label = "OptimizePortfolioMoments"
	if err := validateMoments(label, mean, cov, names); err != nil {
		return nil, err
	}
	copied := make([][]float64, len(cov))
	for i := range cov {
		copied[i] = append([]float64(nil), cov[i]...)
	}
	if names == nil {
		names = defaultAssetNames(len(mean))
	}
	return optimize(label, append([]float64(nil), mean...), copied, append([]string(nil), names...), cfg)
}

// EfficientFrontier returns points target-return portfolios spread evenly
// between the minimum-variance portfolio's expected return and the largest
// return attainable under the bounds, ordered by increasing ExpectedReturn.
// cfg supplies the bounds, tolerance, and risk-free rate; its Objective and
// TargetReturn are ignored because every point is a target-return solve.
//
// Returns an error for points below 2, and every error OptimizePortfolio
// reports.
func EfficientFrontier(returns insyra.IDataTable, points int, cfg PortfolioConfig) ([]PortfolioResult, error) {
	const label = "EfficientFrontier"
	if points < 2 {
		return nil, fmt.Errorf("%s: points must be at least 2, got %d", label, points)
	}
	mean, cov, names, err := portfolioMoments(returns)
	if err != nil {
		return nil, err
	}
	problem, err := newPortfolioProblem(label, mean, cov, cfg)
	if err != nil {
		return nil, err
	}

	base, iterations, converged := problem.solveMinimumVariance(nil)
	low := dotFloat(problem.mean, base)
	_, high := attainableReturnRange(problem.mean, problem.lo, problem.hi)
	if high < low {
		high = low
	}

	results := make([]PortfolioResult, points)
	results[0] = problem.result(base, names, cfg.RiskFreeRate, iterations, converged)
	warm := base
	for i := 1; i < points; i++ {
		target := low + (high-low)*float64(i)/float64(points-1)
		if i == points-1 {
			target = high
		}
		weights, iterations, ok := problem.solveTargetReturn(target, warm)
		results[i] = problem.result(weights, names, cfg.RiskFreeRate, iterations, ok)
		warm = weights
	}
	return results, nil
}

// optimize runs the configured objective on validated moments.
func optimize(label string, mean []float64, cov [][]float64, names []string, cfg PortfolioConfig) (*PortfolioResult, error) {
	problem, err := newPortfolioProblem(label, mean, cov, cfg)
	if err != nil {
		return nil, err
	}

	var (
		weights    []float64
		iterations int
		converged  bool
	)
	switch cfg.Objective {
	case MinimumVariance:
		weights, iterations, converged = problem.solveMinimumVariance(nil)
	case TargetReturn:
		lowest, highest := attainableReturnRange(problem.mean, problem.lo, problem.hi)
		if cfg.TargetReturn < lowest || cfg.TargetReturn > highest {
			return nil, fmt.Errorf("%s: TargetReturn %g is outside the attainable range [%g, %g] under the given bounds", label, cfg.TargetReturn, lowest, highest)
		}
		weights, iterations, converged = problem.solveTargetReturn(cfg.TargetReturn, nil)
	case MaximumSharpe:
		weights, iterations, converged = problem.solveMaximumSharpe(cfg.RiskFreeRate)
	default:
		return nil, fmt.Errorf("%s: unknown Objective %d", label, cfg.Objective)
	}

	result := problem.result(weights, names, cfg.RiskFreeRate, iterations, converged)
	return &result, nil
}

// result packages solved weights with the statistics that describe them.
func (p *portfolioProblem) result(weights []float64, names []string, riskFreeRate float64, iterations int, converged bool) PortfolioResult {
	variance := p.variance(weights)
	volatility := math.Sqrt(variance)
	expected := dotFloat(p.mean, weights)
	sharpe := math.Inf(1)
	if volatility > 0 {
		sharpe = (expected - riskFreeRate) / volatility
	}
	return PortfolioResult{
		Weights:        append([]float64(nil), weights...),
		AssetNames:     names,
		ExpectedReturn: expected,
		Variance:       variance,
		Volatility:     volatility,
		SharpeRatio:    sharpe,
		Iterations:     iterations,
		Converged:      converged,
	}
}

// newPortfolioProblem validates the bounds, scales the covariance so its
// largest eigenvalue is 1, and applies the tolerance and iteration defaults.
func newPortfolioProblem(label string, mean []float64, cov [][]float64, cfg PortfolioConfig) (*portfolioProblem, error) {
	n := len(mean)
	lo, hi, err := resolveBounds(label, n, cfg.MinWeight, cfg.MaxWeight)
	if err != nil {
		return nil, err
	}

	scale := largestEigenvalue(cov)
	if scale <= 0 {
		// A zero covariance leaves the objective flat; scaling by 1 keeps the
		// solver well defined and every feasible point is optimal.
		scale = 1
	}
	scaled := make([][]float64, n)
	for i := range cov {
		scaled[i] = make([]float64, n)
		for j := range cov[i] {
			scaled[i][j] = cov[i][j] / scale
		}
	}

	tolerance := cfg.Tolerance
	if tolerance <= 0 {
		tolerance = 1e-10
	}
	maxIterations := cfg.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10000
	}
	return &portfolioProblem{
		cov:           scaled,
		mean:          mean,
		lo:            lo,
		hi:            hi,
		covScale:      scale,
		tolerance:     tolerance,
		maxIterations: maxIterations,
	}, nil
}

// resolveBounds fills in the default long-only box and refuses bounds that
// cannot hold a portfolio summing to 1.
func resolveBounds(label string, n int, minWeight, maxWeight []float64) ([]float64, []float64, error) {
	lo := make([]float64, n)
	hi := make([]float64, n)
	if minWeight == nil {
		for i := range n {
			lo[i] = 0
		}
	} else {
		if len(minWeight) != n {
			return nil, nil, fmt.Errorf("%s: MinWeight has %d entries for %d assets", label, len(minWeight), n)
		}
		copy(lo, minWeight)
	}
	if maxWeight == nil {
		for i := range n {
			hi[i] = 1
		}
	} else {
		if len(maxWeight) != n {
			return nil, nil, fmt.Errorf("%s: MaxWeight has %d entries for %d assets", label, len(maxWeight), n)
		}
		copy(hi, maxWeight)
	}

	sumLo, sumHi := 0.0, 0.0
	for i := range n {
		if math.IsNaN(lo[i]) || math.IsInf(lo[i], 0) {
			return nil, nil, fmt.Errorf("%s: MinWeight[%d] is not finite", label, i)
		}
		if math.IsNaN(hi[i]) || math.IsInf(hi[i], 0) {
			return nil, nil, fmt.Errorf("%s: MaxWeight[%d] is not finite", label, i)
		}
		if lo[i] > hi[i] {
			return nil, nil, fmt.Errorf("%s: MinWeight[%d] (%g) exceeds MaxWeight[%d] (%g)", label, i, lo[i], i, hi[i])
		}
		sumLo += lo[i]
		sumHi += hi[i]
	}
	if sumLo > 1 {
		return nil, nil, fmt.Errorf("%s: infeasible bounds, MinWeight sums to %g which is above 1", label, sumLo)
	}
	if sumHi < 1 {
		return nil, nil, fmt.Errorf("%s: infeasible bounds, MaxWeight sums to %g which is below 1 (sum(hi) < 1)", label, sumHi)
	}
	return lo, hi, nil
}

// portfolioMoments reads every column of returns and returns the column means,
// the n-1 sample covariance, and the column names.
func portfolioMoments(returns insyra.IDataTable) ([]float64, [][]float64, []string, error) {
	const label = "OptimizePortfolio"
	if returns == nil {
		return nil, nil, nil, fmt.Errorf("%s: returns is nil", label)
	}
	assets := returns.NumCols()
	if assets < 2 {
		return nil, nil, nil, fmt.Errorf("%s: need at least 2 asset columns, got %d", label, assets)
	}
	rawNames := returns.ColNames()
	names := make([]string, assets)
	columns := make([][]float64, assets)
	for j := range assets {
		names[j] = ""
		if j < len(rawNames) {
			names[j] = rawNames[j]
		}
		if names[j] == "" {
			names[j] = fmt.Sprintf("%d", j+1)
		}
		column := returns.GetColByNumber(j)
		if column == nil {
			return nil, nil, nil, fmt.Errorf("%s: column %q is nil", label, names[j])
		}
		values, err := numericSeries(column, fmt.Sprintf("%s: column %s", label, names[j]))
		if err != nil {
			return nil, nil, nil, err
		}
		columns[j] = values
	}

	observations := len(columns[0])
	for j := range assets {
		if len(columns[j]) != observations {
			return nil, nil, nil, fmt.Errorf("%s: column %q has %d rows but column %q has %d", label, names[j], len(columns[j]), names[0], observations)
		}
	}
	if observations < assets+1 {
		return nil, nil, nil, fmt.Errorf("%s: need at least %d observations for %d assets, got %d", label, assets+1, assets, observations)
	}

	mean := make([]float64, assets)
	for j := range assets {
		sum := 0.0
		for _, v := range columns[j] {
			sum += v
		}
		mean[j] = sum / float64(observations)
	}
	cov := make([][]float64, assets)
	for i := range assets {
		cov[i] = make([]float64, assets)
	}
	denominator := float64(observations - 1)
	for i := range assets {
		for j := i; j < assets; j++ {
			sum := 0.0
			for t := range observations {
				sum += (columns[i][t] - mean[i]) * (columns[j][t] - mean[j])
			}
			value := sum / denominator
			cov[i][j] = value
			cov[j][i] = value
		}
	}
	return mean, cov, names, nil
}

// validateMoments checks caller-supplied moments, including the symmetry and
// positive-semidefiniteness the solver assumes.
func validateMoments(label string, mean []float64, cov [][]float64, names []string) error {
	n := len(mean)
	if n < 2 {
		return fmt.Errorf("%s: need at least 2 assets, got %d", label, n)
	}
	if len(cov) != n {
		return fmt.Errorf("%s: cov has %d rows for %d assets", label, len(cov), n)
	}
	if names != nil && len(names) != n {
		return fmt.Errorf("%s: names has %d entries for %d assets", label, len(names), n)
	}
	for i := range n {
		if math.IsNaN(mean[i]) || math.IsInf(mean[i], 0) {
			return fmt.Errorf("%s: mean[%d] is not finite", label, i)
		}
		if len(cov[i]) != n {
			return fmt.Errorf("%s: cov row %d has %d entries for %d assets", label, i, len(cov[i]), n)
		}
		for j := range n {
			if math.IsNaN(cov[i][j]) || math.IsInf(cov[i][j], 0) {
				return fmt.Errorf("%s: cov[%d][%d] is not finite", label, i, j)
			}
		}
	}
	for i := range n {
		for j := i + 1; j < n; j++ {
			scale := math.Max(1, math.Max(math.Abs(cov[i][j]), math.Abs(cov[j][i])))
			if math.Abs(cov[i][j]-cov[j][i]) > 1e-12*scale {
				return fmt.Errorf("%s: cov is not symmetric, cov[%d][%d] = %g but cov[%d][%d] = %g", label, i, j, cov[i][j], j, i, cov[j][i])
			}
		}
	}

	symmetric := mat.NewSymDense(n, nil)
	for i := range n {
		for j := i; j < n; j++ {
			symmetric.SetSym(i, j, 0.5*(cov[i][j]+cov[j][i]))
		}
	}
	var eigen mat.EigenSym
	if !eigen.Factorize(symmetric, false) {
		return fmt.Errorf("%s: cov eigendecomposition failed, so it cannot be checked for positive semidefiniteness", label)
	}
	values := eigen.Values(nil)
	smallest, largest := math.Inf(1), math.Inf(-1)
	for _, v := range values {
		smallest = math.Min(smallest, v)
		largest = math.Max(largest, v)
	}
	if smallest < -1e-10*math.Max(1, largest) {
		return fmt.Errorf("%s: cov is not positive semidefinite, its smallest eigenvalue is %g", label, smallest)
	}
	return nil
}

func defaultAssetNames(n int) []string {
	names := make([]string, n)
	for i := range n {
		names[i] = fmt.Sprintf("%d", i+1)
	}
	return names
}
