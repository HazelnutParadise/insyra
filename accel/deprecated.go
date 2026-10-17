package accel

import "github.com/HazelnutParadise/insyra/internal/acceltest"

func init() {
	acceltest.SetResetDefault(resetDefault)
}

// The names below were exported before the backend-author API and the test
// hooks were made internal (#262). They stay for one release and behave as
// before; none of them is user API.

// BackendExecutor runs an operation on a real device.
//
// Deprecated: backend registration is internal to accel. This alias is removed
// in the next release.
type BackendExecutor = backendExecutor

// ExecuteRequest is one operation over one dataset on one device.
//
// Deprecated: backend registration is internal to accel. This alias is removed
// in the next release.
type ExecuteRequest = executeRequest

// ExecuteResponse carries a backend's results and what the submission cost.
//
// Deprecated: backend registration is internal to accel. This alias is removed
// in the next release.
type ExecuteResponse = executeResponse

// ExecuteColumn is one projected column handed to a backend.
//
// Deprecated: backend registration is internal to accel. This alias is removed
// in the next release.
type ExecuteColumn = executeColumn

// Discoverer finds devices for a session.
//
// Deprecated: device discovery is internal to accel. This alias is removed in
// the next release.
type Discoverer = deviceDiscoverer

// SDKProbe is a backend-specific probe that talks to a vendor SDK.
//
// Deprecated: device discovery is internal to accel. This alias is removed in
// the next release.
type SDKProbe = sdkProbe

// RegisterBackendExecutor makes executor the runner for backend.
//
// Deprecated: backend registration is internal to accel. This function is
// removed in the next release.
func RegisterBackendExecutor(backend Backend, executor BackendExecutor) error {
	return registerBackendExecutor(backend, executor)
}

// RegisterDiscoverer adds d to the discoverers every session consults.
//
// Deprecated: device discovery is internal to accel. This function is removed
// in the next release.
func RegisterDiscoverer(d Discoverer) { registerDiscoverer(d) }

// RegisterSDKProbe adds probe to the SDK probes for its backend.
//
// Deprecated: device discovery is internal to accel. This function is removed
// in the next release.
func RegisterSDKProbe(probe SDKProbe) { registerSDKProbe(probe) }

// RegisterDevice adds device to the session's device list.
//
// Deprecated: a session's devices come from discovery; use Discover. This
// method is removed in the next release.
func (s *Session) RegisterDevice(device Device) error { return s.registerDevice(device) }

// RecordReport appends report to the session's reports.
//
// Deprecated: a session records its own reports; read them with Report or
// Reports. This method is removed in the next release.
func (s *Session) RecordReport(report Report) error { return s.recordReport(report) }

// PlanShardableWorkload plans a shardable workload across the session's
// devices.
//
// Deprecated: planning for a workload is internal to ExecuteNearestExact; use
// PlanShardable to inspect the plan. This method is removed in the next
// release.
func (s *Session) PlanShardableWorkload(workload WorkloadEstimate) ShardPlan {
	return s.planShardableWorkload(workload)
}

// NearestExactCPU is the float64 reference for ExecuteNearestExact.
//
// Deprecated: ExecuteNearestExact returns the same answer with or without a
// device. This function is removed in the next release.
func NearestExactCPU(dataset *Dataset, queries [][]float64, m int) ([]uint32, []float64, int, error) {
	return nearestExactCPU(dataset, queries, m)
}

// ResetBackendExecutorsForTest clears every registered executor.
//
// Deprecated: test hook, removed in the next release.
func ResetBackendExecutorsForTest() { resetBackendExecutors() }

// ResetDefaultForTest drops the shared session so the next Default call
// rediscovers.
//
// Deprecated: test hook, removed in the next release.
func ResetDefaultForTest() { resetDefault() }

// ResetDiscoverersForTest drops every registered discoverer and SDK probe.
//
// Deprecated: test hook, removed in the next release.
func ResetDiscoverersForTest() { resetDiscoverers() }

// ResetSDKProbesForTest drops every registered SDK probe.
//
// Deprecated: test hook, removed in the next release.
func ResetSDKProbesForTest() { resetSDKProbes() }
