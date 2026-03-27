package accel_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/accel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenUsesDefaultsAndExposesReport(t *testing.T) {
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
	require.IsType(t, []int64{}, ds.Buffers[0].Values)
	assert.Equal(t, []int64{1, 2, 0, 4}, ds.Buffers[0].Values)
}

func TestProjectDataTableBuildsOneBufferPerColumn(t *testing.T) {
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

func TestProjectDataListPopulatesCacheSnapshotAndMetrics(t *testing.T) {
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
