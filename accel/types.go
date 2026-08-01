package accel

import "time"

type Mode string

const (
	ModeAuto      Mode = "auto"
	ModeCPU       Mode = "cpu"
	ModeGPU       Mode = "gpu"
	ModeStrictGPU Mode = "strict-gpu"
)

type Backend string

const (
	BackendUnknown Backend = "unknown"
	BackendCPU     Backend = "cpu"
	BackendCUDA    Backend = "cuda"
	BackendMetal   Backend = "metal"
	BackendWebGPU  Backend = "webgpu"
)

type DeviceType string

const (
	DeviceTypeUnknown    DeviceType = "unknown"
	DeviceTypeCPU        DeviceType = "cpu"
	DeviceTypeIntegrated DeviceType = "integrated"
	DeviceTypeDiscrete   DeviceType = "discrete"
	DeviceTypeVirtual    DeviceType = "virtual"
)

type MemoryClass string

const (
	MemoryClassUnknown MemoryClass = "unknown"
	MemoryClassShared  MemoryClass = "shared"
	MemoryClassDevice  MemoryClass = "device-local"
)

type ProbeSource string

const (
	ProbeSourceUnknown ProbeSource = "unknown"
	ProbeSourceSDK     ProbeSource = "sdk"
	ProbeSourceNative  ProbeSource = "native"
	ProbeSourceEnvStub ProbeSource = "env-stub"
)

type DataType string

const (
	DataTypeUnknown DataType = "unknown"
	DataTypeBool    DataType = "bool"
	DataTypeInt64   DataType = "int64"
	DataTypeFloat64 DataType = "float64"
	DataTypeString  DataType = "string"
	DataTypeAny     DataType = "any"
)

type FallbackReason string

const (
	FallbackReasonNone                  FallbackReason = "none"
	FallbackReasonNoAccelerator         FallbackReason = "no-accelerator"
	FallbackReasonCPUOnly               FallbackReason = "cpu-only-mode"
	FallbackReasonDiscoveryError        FallbackReason = "discovery-error"
	FallbackReasonStrictGPUUnavailable  FallbackReason = "strict-gpu-unavailable"
	FallbackReasonWorkloadUnsupported   FallbackReason = "workload-unsupported"
	FallbackReasonWorkloadNotProfitable FallbackReason = "workload-not-profitable"
	FallbackReasonNoBackendExecutor     FallbackReason = "no-backend-executor"
	FallbackReasonPrecisionNotAccepted  FallbackReason = "precision-not-accepted"
	FallbackReasonDTypeNotEligible      FallbackReason = "dtype-not-eligible"
	FallbackReasonShaderCompileFailed   FallbackReason = "shader-compile-failed"
	FallbackReasonBufferTooLarge        FallbackReason = "buffer-too-large"
	FallbackReasonReadbackTimeout       FallbackReason = "readback-timeout"
	FallbackReasonExecutionFailed       FallbackReason = "execution-failed"
)

// Op names the operation a backend is asked to perform. The runtime ships one
// operation; adding a second is a spec change, not a signature change.
type Op string

const (
	OpUnknown Op = "unknown"
	// OpNearestShortlist returns the several nearest query points per row rather
	// than only the nearest, plus the distance of the best rejected one. It is
	// how an exact float64 answer is reached through an f32 device: the device
	// narrows the field, the host settles the ranking.
	//
	// It is the only device operation. Three others existed and were removed once
	// measured: a column sum at 0.7x, a distance matrix whose readback grew with
	// the answer, and a single-precision nearest query no float64 caller could
	// use. Nothing is added back without a measurement against a host using every
	// core it has.
	OpNearestShortlist Op = "nearest-shortlist"
)

// Precision states what the caller will accept from device execution. The
// default refuses anything the device cannot compute at the column's own
// precision, because narrowing a column silently would change the numbers a
// data-analysis library returns. WGSL has no f64 and Apple GPUs have no
// double-precision hardware, so float64 columns need an explicit opt-in.
type Precision string

const (
	PrecisionExact   Precision = "exact"
	PrecisionFloat32 Precision = "float32"
)

type WorkloadClass string

const (
	WorkloadClassUnknown  WorkloadClass = "unknown"
	WorkloadClassColumnar WorkloadClass = "columnar"
)

type MergePolicy string

const (
	MergePolicyUnknown       MergePolicy = "unknown"
	MergePolicyCPU           MergePolicy = "cpu"
	MergePolicyBackendNative MergePolicy = "backend-native"
)

type ExecutorKind string

const (
	ExecutorKindUnknown    ExecutorKind = "unknown"
	ExecutorKindNone       ExecutorKind = "none"
	ExecutorKindRegistered ExecutorKind = "registered"
)

type MemoryBudgetPolicy struct {
	DeviceFraction float64
	SharedFraction float64
}

type Config struct {
	Mode              Mode
	PreferredBackends []Backend
	MemoryBudget      MemoryBudgetPolicy
	Strict            bool
	EnableFallback    bool
	PreferredDevices  []string
	ReportHistorySize int
	DiscoveryTimeout  time.Duration
}

type Device struct {
	ID                string
	Name              string
	Vendor            string
	Backend           Backend
	ProbeSource       ProbeSource
	Type              DeviceType
	MemoryClass       MemoryClass
	SharedMemory      bool
	BudgetBytes       uint64
	Score             float64
	CapabilitySummary map[string]bool
	DriverVersion     string
	ComputeCapability string
	PCIBusID          string
}

type Report struct {
	Mode                Mode
	Accelerated         bool
	SelectedBackend     Backend
	DiscoveredDeviceIDs []string
	SelectedDeviceIDs   []string
	SelectedDevices     []string
	FallbackReason      FallbackReason
	StartedAt           time.Time
	FinishedAt          time.Time
	GeneratedAt         time.Time
	Metrics             map[string]float64
}

func (r Report) Duration() time.Duration {
	if r.StartedAt.IsZero() || r.FinishedAt.IsZero() || r.FinishedAt.Before(r.StartedAt) {
		return 0
	}
	return r.FinishedAt.Sub(r.StartedAt)
}

type Buffer struct {
	Name          string
	Type          DataType
	Values        any
	Nulls         []bool
	Validity      []byte
	StringOffsets []uint32
	StringData    []byte
	Len           int
}

type Dataset struct {
	Name        string
	Fingerprint string
	Lineage     string
	Rows        int
	Buffers     []Buffer
}

type CacheEntry struct {
	Key                 string
	DatasetName         string
	DatasetID           string
	Lineage             string
	BufferName          string
	Type                DataType
	Len                 int
	ResidentBytes       uint64
	DeviceIDs           []string
	DeviceResidentBytes map[string]uint64
	LastAccess          time.Time
	accessOrdinal       uint64
}

type CacheDeviceUsage struct {
	DeviceID        string
	ResidentBuffers int
	ResidentBytes   uint64
	BudgetBytes     uint64
}

type CacheSnapshot struct {
	ResidentBuffers int
	ResidentBytes   uint64
	BudgetBytes     uint64
	EvictedBuffers  uint64
	EvictedBytes    uint64
	DeviceUsage     []CacheDeviceUsage
	Entries         []CacheEntry
}

type WorkloadEstimate struct {
	Class WorkloadClass
	Rows  int
	Bytes uint64
	// Op is the operation to execute on the device. Empty means OpSum.
	Op Op
	// Precision is what the caller will accept. Empty means PrecisionExact.
	Precision Precision
}

type ShardAssignment struct {
	DeviceID     string
	Backend      Backend
	Weight       float64
	SharePercent float64
	Rows         int
	Bytes        uint64
	BudgetBytes  uint64
}

type ExecutionResult struct {
	Accelerated    bool
	FallbackReason FallbackReason
	MergePolicy    MergePolicy
	Executor       string
	ExecutorKind   ExecutorKind
	Assignments    []ShardAssignment
	DeviceIDs      []string

	// Op and Precision describe what actually ran, not what was asked for.
	Op        Op
	Precision Precision

	// Reductions holds one value per buffer, keyed by buffer name. It is only
	// populated when Accelerated is true.
	Reductions map[string]float64
	// Counts holds the number of non-null values folded into each reduction.
	Counts map[string]int

	// Measured cost. These are host-observed durations: Metal and GLES return
	// ErrTimestampsNotSupported for GPU timestamp queries, so only Vulkan and
	// DX12 could report device-side timing. Zero when nothing ran on a device.
	Transfer      time.Duration
	Dispatch      time.Duration
	Readback      time.Duration
	BytesUploaded uint64
}

func DefaultConfig() Config {
	return Config{
		Mode:              ModeAuto,
		PreferredBackends: []Backend{BackendCUDA, BackendMetal, BackendWebGPU},
		MemoryBudget:      MemoryBudgetPolicy{DeviceFraction: 0.60, SharedFraction: 0.35},
		Strict:            false,
		EnableFallback:    true,
		PreferredDevices:  nil,
		ReportHistorySize: 32,
		DiscoveryTimeout:  5 * time.Second,
	}
}
