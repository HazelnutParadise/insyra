package finance

import "testing"

// The last Options used to win, so a caller who passed two never learned the
// first was ignored. More than one is an error, as everywhere else.
func TestMoreThanOneOptionsIsRefused(t *testing.T) {
	rate, pv, fv := mustDec("0.01"), mustDec("1000"), mustDec("0")
	if _, err := PMT(rate, 12, pv, fv, PaymentEnd, Options{Scale: 2}, Options{Scale: 4}); err == nil {
		t.Fatal("PMT accepted two Options")
	}
	if _, err := PMT(rate, 12, pv, fv, PaymentEnd, Options{Scale: 2}); err != nil {
		t.Fatalf("PMT with one Options failed: %v", err)
	}
}
