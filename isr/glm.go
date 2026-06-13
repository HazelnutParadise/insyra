package isr

import (
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

type LogisticRegressionResult = stats.LogisticRegressionResult
type LogisticRegressionOptions = stats.LogisticRegressionOptions
type PoissonRegressionResult = stats.PoissonRegressionResult
type PoissonRegressionOptions = stats.PoissonRegressionOptions
type GLMResult = stats.GLMResult
type GLMOptions = stats.GLMOptions
type GLMFamily = stats.GLMFamily
type GLMLink = stats.GLMLink
type SeparationPolicy = stats.SeparationPolicy
type PredictType = stats.PredictType

const (
	Binomial stats.GLMFamily = stats.Binomial
	Poisson  stats.GLMFamily = stats.Poisson
	Gaussian stats.GLMFamily = stats.Gaussian

	Log      stats.GLMLink = stats.Log
	Logit    stats.GLMLink = stats.Logit
	Identity stats.GLMLink = stats.Identity
	Probit   stats.GLMLink = stats.Probit
	Cloglog  stats.GLMLink = stats.Cloglog

	SepWarn  stats.SeparationPolicy = stats.SepWarn
	SepError stats.SeparationPolicy = stats.SepError
	SepRidge stats.SeparationPolicy = stats.SepRidge

	PredictLinear   stats.PredictType = stats.PredictLinear
	PredictResponse stats.PredictType = stats.PredictResponse
	PredictClass    stats.PredictType = stats.PredictClass
)

func LogisticRegression(y insyra.IDataList, xs ...insyra.IDataList) (*stats.LogisticRegressionResult, error) {
	return stats.LogisticRegression(y, xs...)
}

func LogisticRegressionWithOptions(opts stats.LogisticRegressionOptions, y insyra.IDataList, xs ...insyra.IDataList) (*stats.LogisticRegressionResult, error) {
	return stats.LogisticRegressionWithOptions(opts, y, xs...)
}

func PoissonRegression(y insyra.IDataList, xs ...insyra.IDataList) (*stats.PoissonRegressionResult, error) {
	return stats.PoissonRegression(y, xs...)
}

func PoissonRegressionWithOptions(opts stats.PoissonRegressionOptions, y insyra.IDataList, xs ...insyra.IDataList) (*stats.PoissonRegressionResult, error) {
	return stats.PoissonRegressionWithOptions(opts, y, xs...)
}

func GLM(opts stats.GLMOptions, y insyra.IDataList, xs ...insyra.IDataList) (*stats.GLMResult, error) {
	return stats.GLM(opts, y, xs...)
}
