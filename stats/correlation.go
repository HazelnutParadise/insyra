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

// CorrelationMatrix 計算多個資料列表之間的相關係數矩陣
// dataLists: 一個包含多個 insyra.IDataList 的切片
// method: 計算相關性的方法 (Pearson, Kendall, Spearman)
// 返回一個二維的 float64 切片表示相關係數矩陣，以及一個錯誤物件
func CorrelationMatrix(dataTable insyra.IDataTable, method CorrelationMethod) *insyra.DataTable {
	_, n := dataTable.Size()
	if n < 2 {
		insyra.LogWarning("stats.CorrelationMatrix: Need at least two columns for correlation.")
		return nil
	}

	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			if i == j {
				matrix[i][j] = 1.0 // 變數與自身的相關性為 1
				continue
			}

			corrResult := Correlation(dataTable.GetColByNumber(i), dataTable.GetColByNumber(j), method)
			if corrResult != nil && !math.IsNaN(corrResult.Statistic) {
				matrix[i][j] = corrResult.Statistic
				matrix[j][i] = corrResult.Statistic // 矩陣是對稱的
			} else {
				insyra.LogWarning("stats.CorrelationMatrix: 無法計算列表 %d 和列表 %d 之間的相關性。將設定為 NaN。", i, j)
				matrix[i][j] = math.NaN()
				matrix[j][i] = math.NaN()
			}
		}
	}
	// 將結果轉換為 insyra.DataTable
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
	return dt
}

func Covariance(dlX, dlY insyra.IDataList) float64 {
	meanX := dlX.Mean()
	meanY := dlY.Mean()
	var sum float64 = 0
	for i := range dlX.Len() {
		x := dlX.Data()[i].(float64)
		y := dlY.Data()[i].(float64)
		sum += (x - meanX) * (y - meanY)
	}
	return sum / float64(dlX.Len()-1)
}

type CorrelationResult struct {
	testResultBase
}

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
