package quant

import (
	"math"
	"sort"
)

// This file holds the numerical core of portfolio optimisation: the exact
// Euclidean projection onto the bounded simplex, the power iteration that
// supplies the gradient step, the accelerated projected-gradient loop, the
// augmented Lagrangian that adds the target-return equality, and the
// golden-section search that finds the maximum-Sharpe point of the frontier.
//
// Everything here works on a covariance scaled so that its largest eigenvalue
// is 1. That keeps the stopping test dimensionless — it is measured in weight
// units, which always sum to 1 — so a tolerance means the same thing whatever
// the return frequency or the currency of the input.

// projectBoundedSimplex writes the Euclidean projection of z onto
// {w : sum(w) = 1, lo <= w <= hi} into out.
//
// The projection has the form w(tau) = clamp(z - tau, lo, hi) for one scalar
// tau, and sum(w(tau)) is non-increasing in tau, so tau is found by bisection.
// tau = min(z-hi) clamps every element to hi (sum = sum(hi) >= 1) and
// tau = max(z-lo) clamps every element to lo (sum = sum(lo) <= 1), which
// brackets the root. The caller must have checked that bracket is valid.
func projectBoundedSimplex(z, lo, hi, out []float64) {
	n := len(z)
	if n == 0 {
		return
	}
	low := math.Inf(1)
	high := math.Inf(-1)
	for i := range n {
		if v := z[i] - hi[i]; v < low {
			low = v
		}
		if v := z[i] - lo[i]; v > high {
			high = v
		}
	}
	for range 200 {
		mid := 0.5 * (low + high)
		if mid <= low || mid >= high {
			break
		}
		sum := 0.0
		for i := range n {
			sum += clampFloat(z[i]-mid, lo[i], hi[i])
		}
		if sum > 1 {
			low = mid
		} else {
			high = mid
		}
	}
	tau := 0.5 * (low + high)
	for i := range n {
		out[i] = clampFloat(z[i]-tau, lo[i], hi[i])
	}
}

func clampFloat(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

// largestEigenvalue returns the largest eigenvalue of a symmetric positive
// semidefinite matrix by power iteration. It returns 0 for the zero matrix.
func largestEigenvalue(cov [][]float64) float64 {
	n := len(cov)
	if n == 0 {
		return 0
	}
	v := make([]float64, n)
	next := make([]float64, n)
	for i := range n {
		// A deterministic non-uniform start, so the iteration is not seeded
		// orthogonal to the leading eigenvector.
		v[i] = 1 + 0.25*math.Sin(float64(i+1))
	}
	normalize(v)
	lambda := 0.0
	for range 500 {
		matVec(cov, v, next)
		norm := 0.0
		for _, x := range next {
			norm += x * x
		}
		norm = math.Sqrt(norm)
		if norm == 0 {
			return 0
		}
		for i := range n {
			next[i] /= norm
		}
		if math.Abs(norm-lambda) <= 1e-14*norm {
			lambda = norm
			copy(v, next)
			break
		}
		lambda = norm
		copy(v, next)
	}
	return lambda
}

func normalize(v []float64) {
	norm := 0.0
	for _, x := range v {
		norm += x * x
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return
	}
	for i := range v {
		v[i] /= norm
	}
}

func matVec(m [][]float64, v, out []float64) {
	for i := range m {
		sum := 0.0
		row := m[i]
		for j := range row {
			sum += row[j] * v[j]
		}
		out[i] = sum
	}
}

func dotFloat(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// acceleratedProjectedGradient minimises a convex function whose gradient is
// grad over {w : sum(w) = 1, lo <= w <= hi}, starting from start.
//
// It is FISTA with the adaptive gradient restart of O'Donoghue and Candes:
// momentum is dropped whenever the extrapolated step points against the last
// move, which is what keeps the iteration monotone on badly conditioned
// covariances. lipschitz is an upper bound on the gradient's Lipschitz
// constant and sets the step size.
//
// Convergence is measured by the gradient mapping at a FIXED reference step of
// 1, not at the step actually taken. The two differ once the augmented
// Lagrangian raises lipschitz: a shrinking step would shrink the residual on
// its own and declare convergence that never happened.
func acceleratedProjectedGradient(grad func(w, out []float64), lipschitz float64, lo, hi, start []float64, tolerance float64, maxIterations int) ([]float64, int, bool) {
	n := len(start)
	w := make([]float64, n)
	projectBoundedSimplex(start, lo, hi, w)
	y := append([]float64(nil), w...)

	gradient := make([]float64, n)
	probe := make([]float64, n)
	candidate := make([]float64, n)
	stationary := make([]float64, n)

	step := 1.0 / lipschitz
	momentum := 1.0
	iterations := 0
	for iterations < maxIterations {
		iterations++
		grad(y, gradient)
		for i := range n {
			probe[i] = y[i] - step*gradient[i]
		}
		projectBoundedSimplex(probe, lo, hi, candidate)

		restart := 0.0
		for i := range n {
			restart += (y[i] - candidate[i]) * (candidate[i] - w[i])
		}
		if restart > 0 {
			copy(y, w)
			momentum = 1
			continue
		}

		nextMomentum := 0.5 * (1 + math.Sqrt(1+4*momentum*momentum))
		beta := (momentum - 1) / nextMomentum
		for i := range n {
			y[i] = candidate[i] + beta*(candidate[i]-w[i])
		}
		copy(w, candidate)
		momentum = nextMomentum

		grad(w, gradient)
		for i := range n {
			probe[i] = w[i] - gradient[i]
		}
		projectBoundedSimplex(probe, lo, hi, stationary)
		residual := 0.0
		for i := range n {
			if d := math.Abs(w[i] - stationary[i]); d > residual {
				residual = d
			}
		}
		if residual <= tolerance {
			return w, iterations, true
		}
	}
	return w, iterations, false
}

// portfolioProblem is a validated optimisation problem. cov is scaled so its
// largest eigenvalue is 1; covScale carries the factor back for reporting.
type portfolioProblem struct {
	cov           [][]float64
	mean          []float64
	lo            []float64
	hi            []float64
	covScale      float64
	tolerance     float64
	maxIterations int
}

func (p *portfolioProblem) uniformStart() []float64 {
	n := len(p.mean)
	start := make([]float64, n)
	for i := range n {
		start[i] = 1 / float64(n)
	}
	out := make([]float64, n)
	projectBoundedSimplex(start, p.lo, p.hi, out)
	return out
}

// solveMinimumVariance minimises w'Sigma w over the bounded simplex.
func (p *portfolioProblem) solveMinimumVariance(start []float64) ([]float64, int, bool) {
	if start == nil {
		start = p.uniformStart()
	}
	gradient := func(w, out []float64) { matVec(p.cov, w, out) }
	return acceleratedProjectedGradient(gradient, 1, p.lo, p.hi, start, p.tolerance, p.maxIterations)
}

// returnEqualityTolerance is how close mu'w must come to the requested target
// before the augmented Lagrangian stops. Returns are per period, so this is a
// generous margin under the 1e-10 the specification asks the caller to see.
const returnEqualityTolerance = 1e-12

// solveTargetReturn minimises w'Sigma w subject to mu'w = target on top of the
// bounded simplex, by the method of multipliers: the equality is carried by
// nu*(mu'w - target) + rho/2*(mu'w - target)^2, nu is updated from the
// violation after every inner solve, and rho is raised whenever the violation
// is not shrinking fast enough.
func (p *portfolioProblem) solveTargetReturn(target float64, start []float64) ([]float64, int, bool) {
	if start == nil {
		start = p.uniformStart()
	}
	// At either end of the attainable range the equality pins the portfolio to
	// a single feasible point and the multiplier diverges. The greedy fill IS
	// that point, so it is returned exactly rather than approached.
	lowest, highest := attainableReturnRange(p.mean, p.lo, p.hi)
	endpointTolerance := returnEqualityTolerance * math.Max(1, math.Abs(highest-lowest))
	if target >= highest-endpointTolerance {
		return greedyExtremeWeights(p.mean, p.lo, p.hi, false), 0, true
	}
	if target <= lowest+endpointTolerance {
		return greedyExtremeWeights(p.mean, p.lo, p.hi, true), 0, true
	}
	meanNorm2 := dotFloat(p.mean, p.mean)
	if meanNorm2 == 0 {
		// Every asset has the same (zero) expected return, so the equality is
		// either free or unattainable; attainability was checked by the caller.
		return p.solveMinimumVariance(start)
	}

	// rho is measured against the scaled covariance, whose largest eigenvalue
	// is 1, so rho*|mu|^2 = 1 makes the penalty comparable to the objective.
	rho := 1 / meanNorm2
	maxRho := 1e14 / meanNorm2
	nu := 0.0

	w := append([]float64(nil), start...)
	innerTolerance := math.Max(p.tolerance, 1e-6)
	previousViolation := math.Inf(1)
	total := 0

	for range 80 {
		penalty := rho
		multiplier := nu
		gradient := func(x, out []float64) {
			matVec(p.cov, x, out)
			coefficient := multiplier + penalty*(dotFloat(p.mean, x)-target)
			for i := range out {
				out[i] += coefficient * p.mean[i]
			}
		}
		next, iterations, converged := acceleratedProjectedGradient(gradient, 1+penalty*meanNorm2, p.lo, p.hi, w, innerTolerance, p.maxIterations)
		copy(w, next)
		total += iterations

		violation := dotFloat(p.mean, w) - target
		if math.Abs(violation) <= returnEqualityTolerance && innerTolerance <= p.tolerance && converged {
			return w, total, true
		}
		nu += rho * violation
		if math.Abs(violation) > 0.25*previousViolation && rho < maxRho {
			rho = math.Min(rho*10, maxRho)
		}
		previousViolation = math.Abs(violation)
		innerTolerance = math.Max(p.tolerance, innerTolerance*0.1)
	}
	return w, total, false
}

// attainableReturnRange returns the smallest and largest mu'w reachable on the
// bounded simplex. Both ends are greedy fills: start at the lower bounds and
// pour the remaining weight into the assets with the most (or least) expected
// return until each hits its upper bound.
func attainableReturnRange(mean, lo, hi []float64) (float64, float64) {
	return greedyReturn(mean, lo, hi, true), greedyReturn(mean, lo, hi, false)
}

func greedyReturn(mean, lo, hi []float64, ascending bool) float64 {
	w := greedyExtremeWeights(mean, lo, hi, ascending)
	return dotFloat(mean, w)
}

// greedyExtremeWeights builds the feasible portfolio with the lowest expected
// return (ascending) or the highest (descending).
func greedyExtremeWeights(mean, lo, hi []float64, ascending bool) []float64 {
	n := len(mean)
	w := append([]float64(nil), lo...)
	remaining := 1.0
	for _, v := range lo {
		remaining -= v
	}
	order := make([]int, n)
	for i := range n {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		if ascending {
			return mean[order[a]] < mean[order[b]]
		}
		return mean[order[a]] > mean[order[b]]
	})
	for _, index := range order {
		if remaining <= 0 {
			break
		}
		room := hi[index] - w[index]
		if room <= 0 {
			continue
		}
		add := math.Min(room, remaining)
		w[index] += add
		remaining -= add
	}
	return w
}

// solveMaximumSharpe walks the efficient frontier. Sharpe is unimodal in the
// target return along the frontier, so golden-section search on
// [return of the minimum-variance portfolio, largest attainable return]
// converges without needing derivatives.
func (p *portfolioProblem) solveMaximumSharpe(riskFreeRate float64) ([]float64, int, bool) {
	base, iterations, converged := p.solveMinimumVariance(nil)
	total := iterations
	low := dotFloat(p.mean, base)
	_, high := attainableReturnRange(p.mean, p.lo, p.hi)
	if high <= low+returnEqualityTolerance {
		return base, total, converged
	}

	best := append([]float64(nil), base...)
	bestSharpe := p.sharpe(base, riskFreeRate)
	evaluate := func(target float64) float64 {
		w, iterations, ok := p.solveTargetReturn(target, best)
		total += iterations
		converged = converged && ok
		value := p.sharpe(w, riskFreeRate)
		if value > bestSharpe {
			bestSharpe = value
			copy(best, w)
		}
		return value
	}

	const goldenRatio = 0.6180339887498949
	a, b := low, high
	c := b - goldenRatio*(b-a)
	d := a + goldenRatio*(b-a)
	fc := evaluate(c)
	fd := evaluate(d)
	for range 60 {
		if b-a <= 1e-14*math.Max(1, math.Abs(b)) {
			break
		}
		if fc > fd {
			b, d, fd = d, c, fc
			c = b - goldenRatio*(b-a)
			fc = evaluate(c)
		} else {
			a, c, fc = c, d, fd
			d = a + goldenRatio*(b-a)
			fd = evaluate(d)
		}
	}
	return best, total, converged
}

// sharpe evaluates (mu'w - rf) / sqrt(w'Sigma w) on the ORIGINAL covariance
// scale, so the value matches what the caller will read from the result.
func (p *portfolioProblem) sharpe(w []float64, riskFreeRate float64) float64 {
	variance := p.variance(w)
	if variance <= 0 {
		return math.Inf(1)
	}
	return (dotFloat(p.mean, w) - riskFreeRate) / math.Sqrt(variance)
}

// variance returns w'Sigma w on the original covariance scale.
func (p *portfolioProblem) variance(w []float64) float64 {
	n := len(w)
	total := 0.0
	for i := range n {
		row := p.cov[i]
		inner := 0.0
		for j := range n {
			inner += row[j] * w[j]
		}
		total += w[i] * inner
	}
	total *= p.covScale
	if total < 0 {
		return 0
	}
	return total
}
