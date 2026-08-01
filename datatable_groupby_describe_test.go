package insyra

import "testing"

func TestGroupedDataTableDescribeNumeric(t *testing.T) {
	dt := NewDataTable(
		NewDataList("APAC", "APAC", "EMEA").SetName("region"),
		NewDataList(10, 20, 30).SetName("revenue"),
		NewDataList("retail", "retail", "enterprise").SetName("segment"),
	)

	desc := dt.GroupBy("region").Describe()

	if got := desc.ColNames(); len(got) != 10 || got[0] != "region" || got[1] != "revenue_count" {
		t.Fatalf("unexpected grouped describe columns: %v", got)
	}
	if got := desc.GetColByName("region").Data(); got[0] != "APAC" || got[1] != "EMEA" {
		t.Fatalf("unexpected group order: %v", got)
	}
	if got := desc.GetColByName("revenue_mean").Data(); got[0] != 15.0 || got[1] != 30.0 {
		t.Fatalf("unexpected revenue_mean: %v", got)
	}
}

func TestGroupedDataTableDescribeIncludeAll(t *testing.T) {
	dt := NewDataTable(
		NewDataList("APAC", "APAC", "EMEA").SetName("region"),
		NewDataList("retail", "online", "retail").SetName("segment"),
	)

	desc := dt.GroupBy("region").Describe(DescribeOptions{IncludeAll: true})

	if got := desc.GetColByName("segment_unique").Data(); got[0] != 2 || got[1] != 1 {
		t.Fatalf("unexpected segment_unique: %v", got)
	}
	if got := desc.GetColByName("segment_top").Data(); got[0] != "retail" || got[1] != "retail" {
		t.Fatalf("unexpected segment_top: %v", got)
	}
}
