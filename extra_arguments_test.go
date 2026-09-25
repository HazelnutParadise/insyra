package insyra

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// A trailing ...T or ...XxxOptions stands for one optional value. Given more
// than one, a function used to keep the first and drop the rest without a
// word, so a caller who passed two option sets never learned that one was
// ignored. More than one is now an error everywhere (AGENTS.md, "How a
// function takes settings").
func TestExtraOptionalValuesAreRefused(t *testing.T) {
	list := func() *DataList { return NewDataList(1.0, nil, 3.0, 4.0) }
	table := func() *DataTable {
		return NewDataTable(
			NewDataList(1.0, 2.0, 3.0, 4.0).SetName("x"),
			NewDataList("a", "a", "b", "b").SetName("g"),
		)
	}
	seed := SamplingOptions{UseSeed: true, Seed: 1}

	cases := []struct {
		name string
		run  func() error
	}{
		{"DataList.Shift", func() error { return recorded(list().Shift(1, 0.0, 9.0).Err()) }},
		{"DataTable.ShiftCol", func() error { return recorded(table().ShiftCol("A", 1, 0.0, 9.0).Err()) }},
		{"DataList.FillForward", func() error {
			dl := list()
			dl.FillForward(1, 2)
			if dl.Get(1) != nil {
				t.Error("FillForward filled despite refusing its arguments")
			}
			return recorded(dl.Err())
		}},
		{"DataList.FillBackward", func() error {
			dl := list()
			dl.FillBackward(1, 2)
			if dl.Get(1) != nil {
				t.Error("FillBackward filled despite refusing its arguments")
			}
			return recorded(dl.Err())
		}},
		{"DataList.FillByInterpolation", func() error {
			dl := list()
			dl.FillByInterpolation(true, false)
			if dl.Get(1) != nil {
				t.Error("FillByInterpolation filled despite refusing its arguments")
			}
			return recorded(dl.Err())
		}},
		{"DataList.Rank", func() error { return recorded(NewDataList(3.0, 1.0, 2.0).Rank(true, false).Err()) }},
		{"DataList.Sample", func() error {
			dl := list()
			dl.Sample(2, false, seed, seed)
			return recorded(dl.Err())
		}},
		{"DataList.SampleFrac", func() error {
			dl := list()
			dl.SampleFrac(0.5, false, seed, seed)
			return recorded(dl.Err())
		}},
		{"DataList.Shuffle", func() error {
			dl := list()
			dl.Shuffle(seed, seed)
			return recorded(dl.Err())
		}},
		{"DataTable.Sample", func() error {
			dt := table()
			dt.Sample(2, false, seed, seed)
			return recorded(dt.Err())
		}},
		{"DataTable.SampleFrac", func() error {
			dt := table()
			dt.SampleFrac(0.5, false, seed, seed)
			return recorded(dt.Err())
		}},
		{"DataTable.Shuffle", func() error {
			dt := table()
			dt.Shuffle(seed, seed)
			return recorded(dt.Err())
		}},
		{"DataTable.TrainTestSplit", func() error {
			dt := table()
			dt.TrainTestSplit(0.5, seed, seed)
			return recorded(dt.Err())
		}},
		{"DataList.Describe", func() error {
			dl := list()
			dl.Describe(DescribeOptions{}, DescribeOptions{})
			return recorded(dl.Err())
		}},
		{"DataTable.Describe", func() error {
			dt := table()
			dt.Describe(DescribeOptions{}, DescribeOptions{})
			return recorded(dt.Err())
		}},
		{"GroupedDataTable.Describe", func() error {
			dt := table()
			dt.GroupBy(Name("g")).Describe(DescribeOptions{}, DescribeOptions{})
			return recorded(dt.Err())
		}},
		{"ReadSQL", func() error {
			db := newTestSQLite(t)
			if err := db.Exec("CREATE TABLE t (a INTEGER); INSERT INTO t VALUES (1);").Error; err != nil {
				t.Fatal(err)
			}
			_, err := ReadSQL(db, "t", ReadSQLOptions{}, ReadSQLOptions{})
			return err
		}},
		{"ReadSQLStream", func() error {
			db := newTestSQLite(t)
			if err := db.Exec("CREATE TABLE t (a INTEGER); INSERT INTO t VALUES (1);").Error; err != nil {
				t.Fatal(err)
			}
			for _, err := range ReadSQLStream(context.Background(), db, "t", ReadSQLOptions{}, ReadSQLOptions{}) {
				if err != nil {
					return err
				}
			}
			return nil
		}},
		{"ToSQL", func() error {
			return table().ToSQL(newTestSQLite(t), "out", ToSQLOptions{}, ToSQLOptions{})
		}},
		{"ReadCSV_File", func() error {
			path := filepath.Join(t.TempDir(), "a.csv")
			if err := os.WriteFile(path, []byte("x\n1\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := ReadCSV_File(path, false, true, "utf-8", "big5")
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.run() == nil {
				t.Fatal("more than one optional value was accepted")
			}
		})
	}
}

// recorded turns what Err() returns into an error, keeping nil nil: Err()
// returns *ErrorInfo, and a nil pointer inside an error interface is not nil.
func recorded(info *ErrorInfo) error {
	if info == nil {
		return nil
	}
	return info
}
