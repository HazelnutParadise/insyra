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
	Assignments    []ShardAssignment
	DeviceIDs      []string
	BytesMoved     uint64
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
