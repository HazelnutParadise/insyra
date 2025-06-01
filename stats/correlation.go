package stats

import (
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

// CorrelationMethod 定義了相關係數的計算方法
type CorrelationMethod int

const (
	PearsonCorrelation CorrelationMethod = iota
	KendallCorrelation
	SpearmanCorrelation
)

// CorrelationAnalysis provides a comprehensive correlation analysis.
// It calculates the correlation coefficient matrix, p-value matrix, and overall test (Bartlett's sphericity test) at once.
// Returns: correlation coefficient matrix, p-value matrix, chi-square value, p-value, degrees of freedom.
func CorrelationAnalysis(dataTable insyra.IDataTable, method CorrelationMethod) (*insyra.DataTable, *insyra.DataTable, float64, float64, int) {
	corrMatrix, pMatrix := CorrelationMatrix(dataTable, method)

	// 只有在使用 Pearson 相關係數時執行巴特萊特球形檢定
	var chiSquare, pValue float64
	var df int
	if method == PearsonCorrelation {
		chiSquare, pValue, df = BartlettSphericity(dataTable)
	} else {
		// 對於非 Pearson 相關係數，不計算球形檢定
		chiSquare, pValue, df = math.NaN(), math.NaN(), 0
	}

	return corrMatrix, pMatrix, chiSquare, pValue, df
}

// CorrelationMatrix calculates the correlation coefficient matrix and its corresponding p-value matrix.
// dataTable: The DataTable used to compute the correlation matrix.
// method: The method used to calculate the correlation coefficient (Pearson, Kendall, or Spearman).
// Returns two DataTables: the first contains the correlation coefficient matrix, the second contains the p-value matrix.
func CorrelationMatrix(dataTable insyra.IDataTable, method CorrelationMethod) (corrMatrix *insyra.DataTable, pMatrix *insyra.DataTable) {
	_, n := dataTable.Size()
	if n < 2 {
		insyra.LogWarning("stats.CorrelationMatrixWithP: Need at least two columns for correlation.")
		return nil, nil
	}

	matrix := make([][]float64, n)
	pmatrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
		pmatrix[i] = make([]float64, n)
	}

	for i := range n {
		for j := i; j < n; j++ {
			if i == j {
				matrix[i][j] = 1.0  // 變數與自身的相關性為 1
				pmatrix[i][j] = 0.0 // 變數與自身的 p 值為 0
				continue
			}

			corrResult := Correlation(dataTable.GetColByNumber(i), dataTable.GetColByNumber(j), method)
			if corrResult != nil && !math.IsNaN(corrResult.Statistic) {
				matrix[i][j] = corrResult.Statistic
				matrix[j][i] = corrResult.Statistic // 矩陣是對稱的
				pmatrix[i][j] = corrResult.PValue
				pmatrix[j][i] = corrResult.PValue // p值矩陣也是對稱的
			} else {
				insyra.LogWarning("stats.CorrelationMatrixWithP: Unable to calculate correlation between column %d and column %d. Setting as NaN.", i, j)
				matrix[i][j] = math.NaN()
				matrix[j][i] = math.NaN()
				pmatrix[i][j] = math.NaN()
				pmatrix[j][i] = math.NaN()
			}
		}
	}

	// 將相關係數結果轉換為 insyra.DataTable
	dt := insyra.NewDataTable()
	colNames := insyra.NewDataList()
	for i := range matrix {
		rowName := dataTable.GetColByNumber(i).GetName()
		row := insyra.NewDataList().SetName(rowName)
		colNames.Append(rowName)
		for j := range matrix[i] {
			row.Append(matrix[i][j])
		}
		dt.AppendRowsFromDataList(row)
	}
	dt.AppendRowsFromDataList(colNames)
	dt.SetRowToColNames(-1)

	// 將 p 值結果轉換為 insyra.DataTable
	pdt := insyra.NewDataTable()
	colNames = insyra.NewDataList()
	for i := range pmatrix {
		rowName := dataTable.GetColByNumber(i).GetName()
		row := insyra.NewDataList().SetName(rowName)
		colNames.Append(rowName)
		for j := range pmatrix[i] {
			row.Append(pmatrix[i][j])
		}
		pdt.AppendRowsFromDataList(row)
	}
	pdt.AppendRowsFromDataList(colNames)
	pdt.SetRowToColNames(-1)

	return dt, pdt
}

func Covariance(dlX, dlY insyra.IDataList) float64 {
	meanX := dlX.Mean()
	meanY := dlY.Mean()
	var sum float64 = 0
	for i := 0; i < dlX.Len(); i++ {
		x := dlX.Data()[i].(float64)
		y := dlY.Data()[i].(float64)
		sum += (x - meanX) * (y - meanY)
	}
	return sum / float64(dlX.Len()-1)
}

type CorrelationResult struct {
	testResultBase
}

// Correlation calculates the correlation between two IDataLists using the specified method.
// dlX: the first IDataList.
// dlY: the second IDataList.
// method: the method to use for calculating the correlation (Pearson, Kendall, or Spearman).
// Returns a CorrelationResult containing the correlation statistic, p-value, confidence interval, and degrees of freedom.
func Correlation(dlX, dlY insyra.IDataList, method CorrelationMethod) *CorrelationResult {
	if dlX == nil || dlY == nil || dlX.Len() != dlY.Len() || dlX.Len() < 2 {
		insyra.LogWarning("stats.Correlation: Invalid input length or nil input.")
		return nil
	}
	if dlX.Stdev() == 0 || dlY.Stdev() == 0 {
		insyra.LogWarning("stats.Correlation: Cannot calculate correlation due to zero variance.")
		return nil
	}

	var result CorrelationResult
	switch method {
	case PearsonCorrelation:
		result = pearsonCorrelationWithStats(dlX, dlY)
	case KendallCorrelation:
		result = kendallCorrelationWithStats(dlX, dlY)
	case SpearmanCorrelation:
		result = spearmanCorrelationWithStats(dlX, dlY)
	default:
		insyra.LogWarning("stats.Correlation: Unsupported method.")
		return nil
	}

	if math.IsNaN(result.Statistic) {
		insyra.LogWarning("stats.Correlation: Cannot calculate correlation.")
		return nil
	}

	return &result
}

// BartlettSphericity performs Bartlett's test of sphericity to assess the overall significance of the correlation matrix.
// dataTable: The DataTable containing the data to be tested.
// Returns the chi-square statistic, p-value, and degrees of freedom.
func BartlettSphericity(dataTable insyra.IDataTable) (chiSquare float64, pValue float64, df int) {
	rows, cols := dataTable.Size()
	if cols < 2 {
		insyra.LogWarning("stats.BartlettSphericity: Need at least two columns for sphericity test.")
		return math.NaN(), math.NaN(), 0
	}

	// 建立相關係數矩陣
	corrMatrix := make([][]float64, cols)
	for i := range corrMatrix {
		corrMatrix[i] = make([]float64, cols)
		for j := range corrMatrix[i] {
			if i == j {
				corrMatrix[i][j] = 1.0
			} else {
				col1 := dataTable.GetColByNumber(i)
				col2 := dataTable.GetColByNumber(j)
				result := Correlation(col1, col2, PearsonCorrelation)
				if result != nil && !math.IsNaN(result.Statistic) {
					corrMatrix[i][j] = result.Statistic
				} else {
					corrMatrix[i][j] = 0.0
				}
			}
		}
	}
	// 計算行列式 (使用高斯消元法)
	det := determinantGauss(corrMatrix)
	if det <= 0 {
		insyra.LogWarning("stats.BartlettSphericity: Correlation matrix is singular or not positive definite.")
		return math.NaN(), math.NaN(), 0
	}

	// 計算巴特萊特檢定統計量
	n := float64(rows)
	p := float64(cols)
	chisq := -((n - 1) - (2*p+5)/6) * math.Log(det)
	// 計算自由度
	degreesOfFreedom := int((p * (p - 1)) / 2)

	// 計算 p 值
	pval := 1 - distuv.ChiSquared{K: float64(degreesOfFreedom)}.CDF(chisq)

	return chisq, pval, degreesOfFreedom
}

// determinant 計算方陣的行列式
func determinant(matrix [][]float64) float64 {
	n := len(matrix)
	if n == 1 {
		return matrix[0][0]
	}
	if n == 2 {
		return matrix[0][0]*matrix[1][1] - matrix[0][1]*matrix[1][0]
	}

	// 對較大的矩陣使用拉普拉斯展開
	det := 0.0
	for i := 0; i < n; i++ {
		minor := make([][]float64, n-1)
		for j := range minor {
			minor[j] = make([]float64, n-1)
		}

		for j := 1; j < n; j++ {
			for k := 0; k < n; k++ {
				if k < i {
					minor[j-1][k] = matrix[j][k]
				} else if k > i {
					minor[j-1][k-1] = matrix[j][k]
				}
			}
		}

		sign := 1.0
		if i%2 != 0 {
			sign = -1.0
		}

		det += sign * matrix[0][i] * determinant(minor)
	}

	return det
}

func pearsonCorrelation(dlX, dlY insyra.IDataList) float64 {
	cov := Covariance(dlX, dlY)
	stdX := dlX.Stdev()
	stdY := dlY.Stdev()
	if stdX == 0 || stdY == 0 {
		return math.NaN()
	}
	return cov / (stdX * stdY)
}

func pearsonCorrelationWithStats(dlX, dlY insyra.IDataList) CorrelationResult {
	result := CorrelationResult{}
	corr := pearsonCorrelation(dlX, dlY)
	result.Statistic = corr

	n := float64(dlX.Len())
	if !math.IsNaN(corr) && n > 2 {
		// 使用 Student's t 分布計算 p 值
		tStat := corr * math.Sqrt((n-2)/(1-corr*corr))
		tDist := distuv.StudentsT{
			Mu:    0,
			Sigma: 1,
			Nu:    n - 2,
		}
		pValue := 2 * tDist.CDF(-math.Abs(tStat))
		result.PValue = pValue

		z := 0.5 * math.Log((1+corr)/(1-corr))
		se := 1 / math.Sqrt(n-3)
		zLower := z - 1.96*se
		zUpper := z + 1.96*se
		rLower := (math.Exp(2*zLower) - 1) / (math.Exp(2*zLower) + 1)
		rUpper := (math.Exp(2*zUpper) - 1) / (math.Exp(2*zUpper) + 1)
		ci := [2]float64{rLower, rUpper}
		result.CI = &ci
		df := n - 2
		result.DF = &df
	}
	return result
}

func kendallCorrelationWithStats(dlX, dlY insyra.IDataList) CorrelationResult {
	result := CorrelationResult{}
	x := dlX.ToF64Slice()
	y := dlY.ToF64Slice()
	tau := stat.Kendall(x, y, nil)
	result.Statistic = tau

	n := len(x)
	if n <= 7 {
		perms := generatePermutations(y)
		extreme := 0
		obs := math.Abs(tau)
		for _, perm := range perms {
			altTau := stat.Kendall(x, perm, nil)
			if math.Abs(altTau) >= obs {
				extreme++
			}
		}
		result.PValue = float64(extreme) / float64(len(perms))
	} else {
		z := 3 * tau * math.Sqrt(float64(n*(n-1))) / math.Sqrt(2*float64(2*n+5))
		result.PValue = 2 * (1 - normalCDF(math.Abs(z)))
	}

	return result
}

func generatePermutations(arr []float64) [][]float64 {
	sort.Float64s(arr)
	res := [][]float64{}
	n := len(arr)
	used := make([]bool, n)
	perm := make([]float64, n)

	var dfs func(int)
	dfs = func(depth int) {
		if depth == n {
			copyPerm := make([]float64, n)
			copy(copyPerm, perm)
			res = append(res, copyPerm)
			return
		}
		for i := 0; i < n; i++ {
			if used[i] {
				continue
			}
			used[i] = true
			perm[depth] = arr[i]
			dfs(depth + 1)
			used[i] = false
		}
	}
	dfs(0)
	return res
}

func spearmanCorrelationWithStats(dlX, dlY insyra.IDataList) CorrelationResult {
	result := CorrelationResult{}
	rankX := dlX.Rank()
	rankY := dlY.Rank()
	rho := pearsonCorrelation(rankX, rankY)
	result.Statistic = rho

	n := float64(dlX.Len())
	if math.IsNaN(rho) {
		result.PValue = math.NaN()
		return result
	}

	if n > 2 {
		if math.Abs(rho) >= 0.9999 {
			result.PValue = 0.0
		} else {
			t := rho * math.Sqrt(n-2) / math.Sqrt(1-rho*rho)
			df := n - 2
			tDist := distuv.StudentsT{
				Mu:    0,
				Sigma: 1,
				Nu:    df,
			}
			pValue := 2 * tDist.CDF(-math.Abs(t))
			result.PValue = pValue
			se := 1.0 / math.Sqrt(n-3)
			ci := [2]float64{rho - 1.96*se, rho + 1.96*se}
			result.CI = &ci
			result.DF = &df
		}
	} else {
		result.PValue = math.NaN()
	}

	return result
}

func normalCDF(z float64) float64 {
	return 0.5 * (1 + math.Erf(z/math.Sqrt(2)))
}

// determinantGauss 計算方陣的行列式 (使用高斯消元法)
// 相較於遞歸的拉普拉斯展開法，高斯消元更適合較大的矩陣
func determinantGauss(matrix [][]float64) float64 {
	n := len(matrix)
	if n == 1 {
		return matrix[0][0]
	}
	if n == 2 {
		return matrix[0][0]*matrix[1][1] - matrix[0][1]*matrix[1][0]
	}

	// 創建矩陣的副本，以避免修改原始矩陣
	A := make([][]float64, n)
	for i := range matrix {
		A[i] = make([]float64, n)
		copy(A[i], matrix[i])
	}

	// 高斯消元
	det := 1.0
	for i := 0; i < n; i++ {
		// 找主元
		maxRow := i
		for j := i + 1; j < n; j++ {
			if math.Abs(A[j][i]) > math.Abs(A[maxRow][i]) {
				maxRow = j
			}
		}

		// 如果主元為零，則行列式為零
		if math.Abs(A[maxRow][i]) < 1e-10 {
			return 0.0
		}

		// 如果需要交換行，則將行列式乘以-1
		if maxRow != i {
			A[i], A[maxRow] = A[maxRow], A[i]
			det *= -1
		}

		// 將行列式乘以主元
		det *= A[i][i]

		// 消元過程
		for j := i + 1; j < n; j++ {
			factor := A[j][i] / A[i][i]
			for k := i; k < n; k++ {
				A[j][k] -= factor * A[i][k]
			}
		}
	}

	return det
}
