package insyra

// IDataList defines the behavior expected from a DataList.
type IDataList interface {
	isFragmented() bool
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	updateTimestamp()
	GetName() string
	SetName(string) *DataList
	Data() []any
	Append(values ...any)
	Get(index int) any
	Clone() *DataList
	Count(value any) int
	Counter() map[any]int
	Update(index int, value any)
	InsertAt(index int, value any)
	FindFirst(any) any
	FindLast(any) any
	FindAll(any) []int
	Filter(func(any) bool) *DataList
	ReplaceFirst(any, any)
	ReplaceLast(any, any)
	ReplaceAll(any, any)
	ReplaceOutliers(float64, float64) *DataList
	Pop() any
	Drop(index int) *DataList
	DropAll(...any) *DataList
	DropIfContains(any) *DataList
	Clear() *DataList
	ClearStrings() *DataList
	ClearNumbers() *DataList
	ClearNaNs() *DataList
	ClearOutliers(float64) *DataList
	Normalize() *DataList
	Standardize() *DataList
	FillNaNWithMean() *DataList
	MovingAverage(int) *DataList
	WeightedMovingAverage(int, any) *DataList
	ExponentialSmoothing(float64) *DataList
	DoubleExponentialSmoothing(float64, float64) *DataList
	MovingStdev(int) *DataList
	Len() int
	Sort(ascending ...bool) *DataList
	Map(mapFunc func(int, any) any) *DataList
	Rank() *DataList
	Reverse() *DataList
	Upper() *DataList
	Lower() *DataList
	Capitalize() *DataList // Statistics
	Sum() float64
	Max() float64
	Min() float64
	Mean() float64
	WeightedMean(weights any) float64
	GMean() float64
	Median() float64
	Mode() []float64
	MAD() float64
	Stdev() float64
	StdevP() float64
	Var() float64
	VarP() float64
	Range() float64
	Quartile(int) float64
	IQR() float64
	Percentile(float64) float64
	Difference() *DataList
	Summary()

	// comparison
	IsEqualTo(*DataList) bool
	IsTheSameAs(*DataList) bool
	Show()
	ShowRange(startEnd ...any)
	ShowTypes()
	ShowTypesRange(startEnd ...any)

	// conversion
	ParseNumbers() *DataList
	ParseStrings() *DataList
	ToF64Slice() []float64
	ToStringSlice() []string

	// Interpolation
	LinearInterpolation(float64) float64
	QuadraticInterpolation(float64) float64
	LagrangeInterpolation(float64) float64
	NearestNeighborInterpolation(float64) float64
	NewtonInterpolation(float64) float64
	HermiteInterpolation(float64, []float64) float64
}
