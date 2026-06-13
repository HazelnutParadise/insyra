package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra/stats/internal/parutil"
	"gonum.org/v1/gonum/mat"
)

const (
	defaultIRLSMaxIter      = 25
	defaultIRLSTolerance    = 1e-8
	sepBetaThreshold        = 30.0
	sepSEThreshold          = 1e3
	weightedNormalThreshold = 50_000
)

type irlsOptions struct {
	maxIter   int
	tolerance float64
	offset    []float64
	weights   []float64
	ridge     float64
}

type irlsFit struct {
	beta        []float64
	covUnscaled [][]float64
	eta         []float64
	mu          []float64
	weightsW    []float64
	deviance    float64
	iterations  int
	converged   bool
	sepFlag     bool
}

func fitIRLS(X *mat.Dense, y []float64, fam glmFamily, link glmLink, opts irlsOptions) (*irlsFit, error) {
	if X == nil {
		return nil, errors.New("design matrix is nil")
	}
	n, k := X.Dims()
	if n == 0 || k == 0 {
		return nil, errors.New("design matrix must be non-empty")
	}
	if len(y) != n {
		return nil, fmt.Errorf("y length %d does not match design matrix rows %d", len(y), n)
	}
	maxIter := opts.maxIter
	if maxIter <= 0 {
		maxIter = defaultIRLSMaxIter
	}
	tol := opts.tolerance
	if tol <= 0 {
		tol = defaultIRLSTolerance
	}
	offset := opts.offset
	if offset == nil {
		offset = make([]float64, n)
	} else if len(offset) != n {
		return nil, fmt.Errorf("offset length %d does not match rows %d", len(offset), n)
	}
	weights := opts.weights
	if weights == nil {
		weights = make([]float64, n)
		for i := range weights {
			weights[i] = 1
		}
	} else if len(weights) != n {
		return nil, fmt.Errorf("weights length %d does not match rows %d", len(weights), n)
	}
	for i, w := range weights {
		if w < 0 || math.IsNaN(w) {
			return nil, fmt.Errorf("weights must be non-negative finite values (index %d)", i)
		}
	}

	mu := make([]float64, n)
	eta := make([]float64, n)
	for i := range n {
		mu[i] = fam.initMu(y[i], weights[i])
		eta[i] = link.eta(mu[i])
	}

	z := make([]float64, n)
	workingW := make([]float64, n)
	devOld := math.Inf(1)
	var beta []float64
	dev := math.NaN()
	converged := false
	iterations := 0
	for iter := 1; iter <= maxIter; iter++ {
		for i := range n {
			dmudeta := math.Max(math.Abs(link.muEta(eta[i])), glmSmall)
			v := math.Max(fam.variance(mu[i]), glmSmall)
			workingW[i] = weights[i] * dmudeta * dmudeta / v
			z[i] = (eta[i] - offset[i]) + (y[i]-mu[i])/dmudeta
		}

		var err error
		beta, err = solveWeightedNormalEquations(X, workingW, z, opts.ridge)
		if err != nil {
			return nil, err
		}

		for i := range n {
			etaNoOffset := 0.0
			for j := range k {
				etaNoOffset += X.At(i, j) * beta[j]
			}
			eta[i] = etaNoOffset + offset[i]
			mu[i] = link.mu(eta[i])
		}

		dev = deviance(y, mu, weights, fam)
		if math.IsNaN(dev) || math.IsInf(dev, 0) {
			return nil, errors.New("IRLS produced non-finite deviance")
		}
		iterations = iter
		if math.Abs(dev-devOld)/(math.Abs(dev)+0.1) < tol {
			converged = true
			break
		}
		devOld = dev
	}

	for i := range n {
		dmudeta := math.Max(math.Abs(link.muEta(eta[i])), glmSmall)
		v := math.Max(fam.variance(mu[i]), glmSmall)
		workingW[i] = weights[i] * dmudeta * dmudeta / v
	}
	covUnscaled, err := invertWeightedInformation(X, workingW, opts.ridge)
	if err != nil {
		return nil, err
	}

	return &irlsFit{
		beta:        beta,
		covUnscaled: covUnscaled,
		eta:         append([]float64(nil), eta...),
		mu:          append([]float64(nil), mu...),
		weightsW:    append([]float64(nil), workingW...),
		deviance:    dev,
		iterations:  iterations,
		converged:   converged,
		sepFlag:     detectSeparation(fam, beta, covUnscaled),
	}, nil
}

func fitNullIRLS(y []float64, fam glmFamily, link glmLink, offset, weights []float64) (*irlsFit, error) {
	return fitNullIRLSWithOptions(y, fam, link, offset, weights, irlsOptions{
		maxIter:   defaultIRLSMaxIter,
		tolerance: defaultIRLSTolerance,
	})
}

func fitNullIRLSWithOptions(y []float64, fam glmFamily, link glmLink, offset, weights []float64, opts irlsOptions) (*irlsFit, error) {
	X := mat.NewDense(len(y), 1, nil)
	raw := X.RawMatrix()
	for i := range y {
		raw.Data[i*raw.Stride] = 1
	}
	opts.offset = offset
	opts.weights = weights
	opts.ridge = 0
	return fitIRLS(X, y, fam, link, irlsOptions{
		maxIter:   opts.maxIter,
		tolerance: opts.tolerance,
		offset:    opts.offset,
		weights:   opts.weights,
	})
}

func solveWeightedNormalEquations(X *mat.Dense, w, z []float64, ridge float64) ([]float64, error) {
	xtwx, xtwz := weightedNormalSystem(X, w, z, ridge)
	var betaVec mat.VecDense
	if err := betaVec.SolveVec(xtwx, xtwz); err != nil {
		return nil, fmt.Errorf("weighted normal equations are singular: %w", err)
	}
	k := betaVec.Len()
	beta := make([]float64, k)
	for i := range k {
		beta[i] = betaVec.AtVec(i)
	}
	return beta, nil
}

func invertWeightedInformation(X *mat.Dense, w []float64, ridge float64) ([][]float64, error) {
	_, k := X.Dims()
	z := make([]float64, len(w))
	xtwx, _ := weightedNormalSystem(X, w, z, ridge)
	var inv mat.Dense
	if err := inv.Inverse(xtwx); err != nil {
		return nil, fmt.Errorf("weighted information matrix is singular: %w", err)
	}
	out := make([][]float64, k)
	for i := range k {
		out[i] = make([]float64, k)
		for j := range k {
			out[i][j] = inv.At(i, j)
		}
	}
	return out, nil
}

func weightedNormalSystem(X *mat.Dense, w, z []float64, ridge float64) (*mat.Dense, *mat.VecDense) {
	n, k := X.Dims()
	xtwxData := make([]float64, k*k)
	xtwzData := make([]float64, k)
	raw := X.RawMatrix()
	stride := raw.Stride

	goParallel := n*k*k >= weightedNormalThreshold
	if !goParallel {
		accumulateWeightedNormal(raw.Data, stride, n, k, w, z, xtwxData, xtwzData)
	} else {
		workers := parutil.MaxWorkers(n)
		partX := make([][]float64, workers)
		partZ := make([][]float64, workers)
		done := make(chan int, workers)
		for worker := range workers {
			start, end := parutil.ChunkBounds(n, workers, worker)
			partX[worker] = make([]float64, k*k)
			partZ[worker] = make([]float64, k)
			go func(worker, start, end int) {
				accumulateWeightedNormal(raw.Data[start*stride:], stride, end-start, k, w[start:end], z[start:end], partX[worker], partZ[worker])
				done <- worker
			}(worker, start, end)
		}
		for range workers {
			worker := <-done
			for i := range xtwxData {
				xtwxData[i] += partX[worker][i]
			}
			for i := range xtwzData {
				xtwzData[i] += partZ[worker][i]
			}
		}
	}

	if ridge > 0 {
		for j := 1; j < k; j++ {
			xtwxData[j*k+j] += ridge
		}
	}

	return mat.NewDense(k, k, xtwxData), mat.NewVecDense(k, xtwzData)
}

func accumulateWeightedNormal(xData []float64, stride, n, k int, w, z, xtwx, xtwz []float64) {
	for i := range n {
		wi := w[i]
		if wi <= 0 {
			continue
		}
		base := i * stride
		wz := wi * z[i]
		for a := range k {
			xa := xData[base+a]
			xtwz[a] += xa * wz
			wxa := wi * xa
			for b := 0; b <= a; b++ {
				xtwx[a*k+b] += wxa * xData[base+b]
			}
		}
	}
	for a := range k {
		for b := 0; b < a; b++ {
			xtwx[b*k+a] = xtwx[a*k+b]
		}
	}
}

func deviance(y, mu, weights []float64, fam glmFamily) float64 {
	sum := 0.0
	for i := range y {
		sum += fam.devianceResidualSq(y[i], mu[i], weights[i])
	}
	return sum
}

func detectSeparation(fam glmFamily, beta []float64, covUnscaled [][]float64) bool {
	if fam.name() != string(Binomial) {
		return false
	}
	for _, b := range beta {
		if math.Abs(b) > sepBetaThreshold {
			return true
		}
	}
	for i := range covUnscaled {
		if i < len(covUnscaled[i]) && covUnscaled[i][i] > 0 && math.Sqrt(covUnscaled[i][i]) > sepSEThreshold {
			return true
		}
	}
	return false
}
