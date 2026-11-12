package datafetch

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	twcbOvernightCSV = "\xef\xbb\xbf\"日\",\"隔夜-加權平均-原始值(%)\",\"隔夜-最高-原始值(%)\",\"隔夜-最低-原始值(%)\",\"1週-加權平均-原始值(%)\"\n" +
		"\"20010119\",\"4.636\",\"5.000\",\"4.615\",\"4.8\"\n" +
		"\"20010120\",\"-\",\"-\",\"-\",\"-\"\n" +
		"\"20010100\",\"-\",\"-\",\"-\",\"-\"\n" +
		"\"20010122\",\"4.600\",\"4.700\",\"4.500\",\"4.8\"\n"
	twcbFiveBankHeader = "\xef\xbb\xbf銀行,年月,活期存款-固定,定存利率-一年期-固定,定存利率-一年期-機動\n"
)

func twcbFiveBankCSV(rows ...string) string {
	return twcbFiveBankHeader + strings.Join(rows, "\n") + "\n"
}

func newFixtureTWCentralBank(t *testing.T, path, body string) (*TWCentralBankClient, *fixtureTransport) {
	t.Helper()
	client, err := TWCentralBank(TWStockConfig{})
	if err != nil {
		t.Fatal(err)
	}
	transport := newFixtureTransport()
	client.base.client.Transport = transport
	client.baseURL = "https://cbc.test"
	u, err := url.Parse("https://cbc.test" + path)
	if err != nil {
		t.Fatal(err)
	}
	key := u.EscapedPath()
	transport.replies[key] = []fixtureReply{{status: http.StatusOK, body: []byte(body)}}
	return client, transport
}

func TestTWCentralBankOvernightRate(t *testing.T) {
	client, _ := newFixtureTWCentralBank(t, twcbOvernightPath, twcbOvernightCSV)
	dt, err := client.OvernightRate(dateUTC(2001, time.January, 19), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.ColNames(); !reflect.DeepEqual(got, []string{"Date", "Rate", "High", "Low"}) {
		t.Fatalf("columns = %v", got)
	}
	want := [][]any{
		{dateUTC(2001, time.January, 19), 4.636, 5.0, 4.615},
		{dateUTC(2001, time.January, 20), nil, nil, nil},
		{dateUTC(2001, time.January, 22), 4.6, 4.7, 4.5},
	}
	if dt.NumRows() != len(want) {
		t.Fatalf("rows = %d, want %d (the month summary row is dropped)", dt.NumRows(), len(want))
	}
	for r, row := range want {
		if got := dt.GetRow(r).Data(); !reflect.DeepEqual(got, row) {
			t.Errorf("row %d = %v, want %v", r, got, row)
		}
	}
}

func TestTWCentralBankOvernightRateRange(t *testing.T) {
	client, _ := newFixtureTWCentralBank(t, twcbOvernightPath, twcbOvernightCSV)
	dt, err := client.OvernightRate(dateUTC(2001, time.January, 20), dateUTC(2001, time.January, 21))
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 1 {
		t.Errorf("rows = %d, want 1", dt.NumRows())
	}
}

func TestTWCentralBankDepositRate(t *testing.T) {
	five := func(month, changhua, land string) []string {
		rows := []string{
			"臺灣銀行," + month + ",,1.000,0.9",
			"合作金庫," + month + ",,1.001,0.9",
			"第一銀行," + month + ",,1.002,0.9",
			"華南銀行," + month + ",,1.003,0.9",
		}
		if changhua != "" {
			rows = append(rows, "彰化銀行,"+month+",,"+changhua+",0.9")
		}
		if land != "" {
			rows = append(rows, "土地銀行,"+month+",,"+land+",0.9")
		}
		return rows
	}
	var rows []string
	rows = append(rows, five("09710", "1.004", "")...)
	rows = append(rows, five("09711", "9.000", "1.009")...) // both banks in the month of the change
	rows = append(rows, five("09712", "", "1.009")...)
	rows = append(rows, "土地銀行,11412,,1.700,1.7") // one bank only
	client, _ := newFixtureTWCentralBank(t, twcbFiveBankPath, twcbFiveBankCSV(rows...))

	dt, err := client.DepositRate(dateUTC(2008, time.October, 20), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.ColNames(); !reflect.DeepEqual(got, []string{"Month", "Rate"}) {
		t.Fatalf("columns = %v", got)
	}
	want := [][]any{
		{dateUTC(2008, time.October, 1), 1.002},
		{dateUTC(2008, time.November, 1), 1.003},
		{dateUTC(2008, time.December, 1), 1.003},
		{dateUTC(2025, time.December, 1), nil},
	}
	if dt.NumRows() != len(want) {
		t.Fatalf("rows = %d, want %d", dt.NumRows(), len(want))
	}
	for r, row := range want {
		if got := dt.GetRow(r).Data(); !reflect.DeepEqual(got, row) {
			t.Errorf("row %d = %v, want %v", r, got, row)
		}
	}
}

func TestTWCentralBankFailures(t *testing.T) {
	client, transport := newFixtureTWCentralBank(t, twcbOvernightPath, twcbOvernightCSV)
	if _, err := client.OvernightRate(dateUTC(2001, time.February, 1), dateUTC(2001, time.January, 1)); err == nil {
		t.Error("to before from: no error")
	}
	if len(transport.requests) != 0 {
		t.Errorf("requests = %d, want none", len(transport.requests))
	}
	bad, _ := newFixtureTWCentralBank(t, twcbOvernightPath, "\"日\",\"隔夜\"\n\"20010119\",\"4.6\"\n")
	if _, err := bad.OvernightRate(time.Time{}, time.Time{}); err == nil || !strings.Contains(err.Error(), "隔夜-加權平均") {
		t.Errorf("changed headers: err = %v", err)
	}
	month, _ := newFixtureTWCentralBank(t, twcbFiveBankPath, twcbFiveBankCSV("臺灣銀行,abc,,1.0,1.0"))
	if _, err := month.DepositRate(time.Time{}, time.Time{}); err == nil || !strings.Contains(err.Error(), `"abc"`) {
		t.Errorf("bad month: err = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.DepositRateContext(ctx, time.Time{}, time.Time{}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: err = %v", err)
	}
	var zero TWCentralBankClient
	if _, err := zero.OvernightRate(time.Time{}, time.Time{}); !errors.Is(err, errTWCentralBankNotCreated) {
		t.Errorf("zero client: err = %v", err)
	}
}
