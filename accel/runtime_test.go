package accel_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/accel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func disableNativeProbesForRuntimeTest(t *testing.T) {
	t.Helper()
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
}

func TestOpenUsesDefaultsAndExposesReport(t *testing.T) {
	disableNativeProbesForRuntimeTest(t)
	session, err := accel.Open(accel.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, session.Close())
	})

	assert.Equal(t, accel.ModeAuto, session.Config().Mode)
	assert.Empty(t, session.Devices())

	report := session.Report()
	assert.Equal(t, accel.ModeAuto, report.Mode)
	assert.Equal(t, accel.FallbackReasonNoAccelerator, report.FallbackReason)
	assert.False(t, report.Accelerated)
}

func TestProjectDataListBuildsTypedDataset(t *testing.T) {
	disableNativeProbesForRuntimeTest(t)
	session, err := accel.Open(accel.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, session.Close())
	})

	dl := insyra.NewDataList(1, 2, nil, 4).SetName("numbers")

	ds, err := session.ProjectDataList(dl)
	require.NoError(t, err)

	require.Equal(t, 1, len(ds.Buffers))
	assert.Equal(t, "numbers", ds.Buffers[0].Name)
	assert.Equal(t, accel.DataTypeInt64, ds.Buffers[0].Type)
	assert.Equal(t, 4, ds.Rows)
	assert.Equal(t, []bool{false, false, true, false}, ds.Buffers[0].Nulls)
	assert.Equal(t, []byte{0b00001011}, ds.Buffers[0].Validity)
	require.IsType(t, []int64{}, ds.Buffers[0].Values)
	assert.Equal(t, []int64{1, 2, 0, 4}, ds.Buffers[0].Values)
}

func TestProjectDataTableBuildsOneBufferPerColumn(t *testing.T) {
	disableNativeProbesForRuntimeTest(t)
	session, err := accel.Open(accel.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, session.Close())
	})

	dt := insyra.NewDataTable(
		insyra.NewDataList("a", "b").SetName("label"),
		insyra.NewDataList(1.5, nil).SetName("score"),
	).SetName("sample")

	ds, err := session.ProjectDataTable(dt)
	require.NoError(t, err)

	assert.Equal(t, "sample", ds.Name)
	assert.Equal(t, 2, ds.Rows)
	require.Len(t, ds.Buffers, 2)
	assert.Equal(t, accel.DataTypeString, ds.Buffers[0].Type)
	assert.Equal(t, accel.DataTypeFloat64, ds.Buffers[1].Type)
	assert.Equal(t, []bool{false, true}, ds.Buffers[1].Nulls)
}

func TestProjectDataListBuildsEncodedStringTransport(t *testing.T) {
	disableNativeProbesForRuntimeTest(t)
	session, err := accel.Open(accel.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, session.Close())
	})

	dl := insyra.NewDataList("ab", nil, "xyz").SetName("labels")

	ds, err := session.ProjectDataList(dl)
	require.NoError(t, err)
	require.Len(t, ds.Buffers, 1)

	buf := ds.Buffers[0]
	assert.Equal(t, accel.DataTypeString, buf.Type)
	assert.Equal(t, []bool{false, true, false}, buf.Nulls)
	assert.Equal(t, []byte{0b00000101}, buf.Validity)
	assert.Equal(t, []uint32{0, 2, 2, 5}, buf.StringOffsets)
	assert.Equal(t, []byte("abxyz"), buf.StringData)
	require.IsType(t, []string{}, buf.Values)
	assert.Equal(t, []string{"ab", "", "xyz"}, buf.Values)
}

func TestProjectDataListPopulatesCacheSnapshotAndMetrics(t *testing.T) {
	disableNativeProbesForRuntimeTest(t)
	session, err := accel.Open(accel.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, session.Close())
	})

	dl := insyra.NewDataList(1, 2, nil, 4).SetName("numbers")

	_, err = session.ProjectDataList(dl)
	require.NoError(t, err)

	snapshot := session.CacheSnapshot()
	assert.Equal(t, 1, snapshot.ResidentBuffers)
	assert.Greater(t, snapshot.ResidentBytes, uint64(0))
	require.Len(t, snapshot.Entries, 1)
	assert.Equal(t, "numbers", snapshot.Entries[0].BufferName)
	assert.Equal(t, float64(1), session.Report().Metrics["cache.resident_buffers"])
	assert.Greater(t, session.Report().Metrics["cache.resident_bytes"], float64(0))
}

func TestProjectedCacheRemainsSessionLocalUntilDeviceResidencyExists(t *testing.T) {
	disableNativeProbesForRuntimeTest(t)
	accel.ResetDiscoverersForTest()
	t.Cleanup(accel.ResetDiscoverersForTest)

	accel.RegisterDiscoverer(runtimeStubDiscoverer{
		name: "mixed-devices",
		devices: []accel.Device{
			{
				ID:          "cuda:test:0",
				Backend:     accel.BackendCUDA,
				Type:        accel.DeviceTypeDiscrete,
				MemoryClass: accel.MemoryClassDevice,
				BudgetBytes: 1024,
				Score:       100,
			},
			{
				ID:           "webgpu:test:0",
				Backend:      accel.BackendWebGPU,
				Type:         accel.DeviceTypeIntegrated,
				MemoryClass:  accel.MemoryClassShared,
				SharedMemory: true,
				BudgetBytes:  1024,
				Score:        90,
			},
		},
	})

	session, err := accel.Open(accel.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, session.Close())
	})

	_, err = session.ProjectDataList(insyra.NewDataList(1, 2, nil, 4).SetName("numbers"))
	require.NoError(t, err)

	snapshot := session.CacheSnapshot()
	require.Len(t, snapshot.Entries, 1)
	assert.Empty(t, snapshot.Entries[0].DeviceIDs)
	for _, usage := range snapshot.DeviceUsage {
		assert.Zero(t, usage.ResidentBuffers)
		assert.Zero(t, usage.ResidentBytes)
	}
	assert.Equal(t, snapshot.Entries[0].ResidentBytes, snapshot.ResidentBytes)
}

type runtimeStubDiscoverer struct {
	name    string
	devices []accel.Device
	err     error
}

func (d runtimeStubDiscoverer) Name() string { return d.name }

func (d runtimeStubDiscoverer) Discover(cfg accel.Config) ([]accel.Device, error) {
	return d.devices, d.err
}

func TestProjectDataListEnforcesCacheBudget(t *testing.T) {
	disableNativeProbesForRuntimeTest(t)
	accel.ResetDiscoverersForTest()
	t.Cleanup(accel.ResetDiscoverersForTest)

	accel.RegisterDiscoverer(runtimeStubDiscoverer{
		name: "tiny-cuda",
		devices: []accel.Device{
			{
				ID:          "cuda:tiny:0",
				Backend:     accel.BackendCUDA,
				Type:        accel.DeviceTypeDiscrete,
				MemoryClass: accel.MemoryClassDevice,
				BudgetBytes: 100,
				Score:       100,
			},
		},
	})

	session, err := accel.Open(accel.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, session.Close())
	})

	_, err = session.ProjectDataList(insyra.NewDataList(1, 2, nil, 4).SetName("first"))
	require.NoError(t, err)
	_, err = session.ProjectDataList(insyra.NewDataList(5, 6, nil, 8).SetName("second"))
	require.NoError(t, err)

	snapshot := session.CacheSnapshot()
	assert.Equal(t, uint64(60), snapshot.BudgetBytes)
	assert.LessOrEqual(t, snapshot.ResidentBytes, snapshot.BudgetBytes)
	assert.Equal(t, 1, snapshot.ResidentBuffers)
	require.Len(t, snapshot.Entries, 1)
	assert.Equal(t, "second", snapshot.Entries[0].BufferName)
	assert.Equal(t, float64(1), session.Report().Metrics["cache.evicted_buffers"])
	assert.Greater(t, session.Report().Metrics["cache.evicted_bytes"], float64(0))
}

func TestExecuteDataListProjectsAndRuns(t *testing.T) {
	disableNativeProbesForRuntimeTest(t)
	accel.ResetDiscoverersForTest()
	t.Cleanup(accel.ResetDiscoverersForTest)

	accel.RegisterDiscoverer(runtimeStubDiscoverer{
		name: "exec-devices",
		devices: []accel.Device{
			{
				ID:          "cuda:exec:0",
				Backend:     accel.BackendCUDA,
				Type:        accel.DeviceTypeDiscrete,
				MemoryClass: accel.MemoryClassDevice,
				BudgetBytes: 1 << 20,
				Score:       100,
			},
			{
				ID:           "webgpu:exec:0",
				Backend:      accel.BackendWebGPU,
				Type:         accel.DeviceTypeIntegrated,
				MemoryClass:  accel.MemoryClassShared,
				SharedMemory: true,
				BudgetBytes:  1 << 20,
				Score:        80,
			},
		},
	})

	session, err := accel.Open(accel.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, session.Close())
	})

	values := make([]any, 512)
	for i := range values {
		values[i] = i + 1
	}

	result, err := session.ExecuteDataList(insyra.NewDataList(values...).SetName("numbers"), accel.WorkloadEstimate{})
	require.NoError(t, err)
	assert.True(t, result.Accelerated)
	assert.Len(t, result.Assignments, 2)

	snapshot := session.CacheSnapshot()
	require.Len(t, snapshot.Entries, 1)
	assert.Equal(t, "numbers", snapshot.Entries[0].BufferName)
	assert.Len(t, snapshot.Entries[0].DeviceResidentBytes, 2)
}
