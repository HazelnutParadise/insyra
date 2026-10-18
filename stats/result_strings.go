package stats

import "fmt"

// This file gives every result type its text form. Each String is
// formatResult with the type's title, so every result prints by the rules
// result_text.go describes; each Show prints String to standard output.

// Hypothesis tests.

// String returns the result as text: "Hypothesis test", then one line per field.
func (r *TestResult) String() string { return formatResult("Hypothesis test", r) }

// Show prints String to standard output.
func (r *TestResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "t-test", then one line per field.
func (r *TTestResult) String() string { return formatResult("t-test", r) }

// Show prints String to standard output.
func (r *TTestResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "z-test", then one line per field.
func (r *ZTestResult) String() string { return formatResult("z-test", r) }

// Show prints String to standard output.
func (r *ZTestResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "F-test", then one line per field.
func (r *FTestResult) String() string { return formatResult("F-test", r) }

// Show prints String to standard output.
func (r *FTestResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Chi-square test", then one line per field.
func (r *ChiSquareTestResult) String() string { return formatResult("Chi-square test", r) }

// Show prints String to standard output.
func (r *ChiSquareTestResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Correlation test", then one line per field.
func (r *CorrelationResult) String() string { return formatResult("Correlation test", r) }

// Show prints String to standard output.
func (r *CorrelationResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Wilcoxon signed-rank test", then one line per field.
func (r *WilcoxonTestResult) String() string { return formatResult("Wilcoxon signed-rank test", r) }

// Show prints String to standard output.
func (r *WilcoxonTestResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Mann-Whitney U test", then one line per field.
func (r *MannWhitneyUResult) String() string { return formatResult("Mann-Whitney U test", r) }

// Show prints String to standard output.
func (r *MannWhitneyUResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Kruskal-Wallis test", then one line per field.
func (r *KruskalWallisResult) String() string { return formatResult("Kruskal-Wallis test", r) }

// Show prints String to standard output.
func (r *KruskalWallisResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Friedman test", then one line per field.
func (r *FriedmanTestResult) String() string { return formatResult("Friedman test", r) }

// Show prints String to standard output.
func (r *FriedmanTestResult) Show() { fmt.Println(r.String()) }

// ANOVA.

// String returns the result as text: "One-way ANOVA", then one line per field.
func (r *OneWayANOVAResult) String() string { return formatResult("One-way ANOVA", r) }

// Show prints String to standard output.
func (r *OneWayANOVAResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Two-way ANOVA", then one line per field.
func (r *TwoWayANOVAResult) String() string { return formatResult("Two-way ANOVA", r) }

// Show prints String to standard output.
func (r *TwoWayANOVAResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Repeated-measures ANOVA", then one line per field.
func (r *RepeatedMeasuresANOVAResult) String() string {
	return formatResult("Repeated-measures ANOVA", r)
}

// Show prints String to standard output.
func (r *RepeatedMeasuresANOVAResult) Show() { fmt.Println(r.String()) }

// Decomposition and clustering.

// String returns the result as text: "PCA", then one line per field.
func (r *PCAResult) String() string { return formatResult("PCA", r) }

// Show prints String to standard output.
func (r *PCAResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "K-means clustering", then one line per field.
func (r *KMeansResult) String() string { return formatResult("K-means clustering", r) }

// Show prints String to standard output.
func (r *KMeansResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Hierarchical clustering", then one line per field.
func (r *HierarchicalResult) String() string { return formatResult("Hierarchical clustering", r) }

// Show prints String to standard output.
func (r *HierarchicalResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "DBSCAN clustering", then one line per field.
func (r *DBSCANResult) String() string { return formatResult("DBSCAN clustering", r) }

// Show prints String to standard output.
func (r *DBSCANResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Silhouette", then one line per field.
func (r *SilhouetteResult) String() string { return formatResult("Silhouette", r) }

// Show prints String to standard output.
func (r *SilhouetteResult) Show() { fmt.Println(r.String()) }

// KNN.

// String returns the result as text: "KNN classification", then one line per field.
func (r *KNNClassificationResult) String() string { return formatResult("KNN classification", r) }

// Show prints String to standard output.
func (r *KNNClassificationResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "KNN regression", then one line per field.
func (r *KNNRegressionResult) String() string { return formatResult("KNN regression", r) }

// Show prints String to standard output.
func (r *KNNRegressionResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "KNN neighbors", then one line per field.
func (r *KNNNeighborsResult) String() string { return formatResult("KNN neighbors", r) }

// Show prints String to standard output.
func (r *KNNNeighborsResult) Show() { fmt.Println(r.String()) }

// Regression.

// String returns the result as text: "Linear regression", then one line per field.
func (r *LinearRegressionResult) String() string { return formatResult("Linear regression", r) }

// Show prints String to standard output.
func (r *LinearRegressionResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Polynomial regression", then one line per field.
func (r *PolynomialRegressionResult) String() string { return formatResult("Polynomial regression", r) }

// Show prints String to standard output.
func (r *PolynomialRegressionResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Exponential regression", then one line per field.
func (r *ExponentialRegressionResult) String() string {
	return formatResult("Exponential regression", r)
}

// Show prints String to standard output.
func (r *ExponentialRegressionResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Logarithmic regression", then one line per field.
func (r *LogarithmicRegressionResult) String() string {
	return formatResult("Logarithmic regression", r)
}

// Show prints String to standard output.
func (r *LogarithmicRegressionResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Generalized linear model", then one line per field.
func (r *GLMResult) String() string { return formatResult("Generalized linear model", r) }

// Show prints String to standard output.
func (r *GLMResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Logistic regression", then one line per field.
func (r *LogisticRegressionResult) String() string { return formatResult("Logistic regression", r) }

// Show prints String to standard output.
func (r *LogisticRegressionResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Poisson regression", then one line per field.
func (r *PoissonRegressionResult) String() string { return formatResult("Poisson regression", r) }

// Show prints String to standard output.
func (r *PoissonRegressionResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Ridge regression", then one line per field.
func (r *RidgeRegressionResult) String() string { return formatResult("Ridge regression", r) }

// Show prints String to standard output.
func (r *RidgeRegressionResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Lasso regression", then one line per field.
func (r *LassoRegressionResult) String() string { return formatResult("Lasso regression", r) }

// Show prints String to standard output.
func (r *LassoRegressionResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Weighted linear regression", then one line per field.
func (r *WeightedLinearRegressionResult) String() string {
	return formatResult("Weighted linear regression", r)
}

// Show prints String to standard output.
func (r *WeightedLinearRegressionResult) Show() { fmt.Println(r.String()) }

// Factor analysis.

// String returns the result as text: "Bartlett's test of sphericity", then one line per field.
func (r *BartlettTestResult) String() string { return formatResult("Bartlett's test of sphericity", r) }

// Show prints String to standard output.
func (r *BartlettTestResult) Show() { fmt.Println(r.String()) }

// String returns the result as text: "Factor analysis", then one line per field.
func (r *FactorAnalysisResult) String() string { return formatResult("Factor analysis", r) }

// String returns the model's result as text, the same text its
// FactorAnalysisResult gives.
func (m *FactorModel) String() string {
	if m == nil {
		return "<nil>"
	}
	return m.FactorAnalysisResult.String()
}

// Show prints the model the way FactorAnalysisResult.Show does.
func (m *FactorModel) Show(startEndRange ...any) {
	if m == nil {
		fmt.Println("<nil>")
		return
	}
	m.FactorAnalysisResult.Show(startEndRange...)
}
