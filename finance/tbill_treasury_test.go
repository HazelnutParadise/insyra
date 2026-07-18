package finance

import "testing"

// TBillEq follows the US Treasury / SIA coupon-equivalent ("investment rate")
// convention: the simple money-market formula for DSM <= 182 and the long-bill
// quadratic for DSM > 182. The want values below are the Treasury/SIA formula
// computed independently in Python and cross-checked against QuantLib's exact
// semi-annually compounded yield (agreement within ~1 bp for DSM > 182); they
// deliberately differ from Excel's TBILLEQ, which keeps using the simple formula
// past 182 days and therefore overstates the long-bill yield.
func TestTBillEq_TreasuryInvestmentRate(t *testing.T) {
	settle := date(2023, 1, 15)
	cases := []struct {
		dsm  int
		want string
	}{
		{62, "0.0511347716"},  // short bill: == Excel TBILLEQ
		{182, "0.0520091194"}, // boundary: still the simple formula
		{183, "0.0520128355"}, // just past boundary: continuous, numerically stable
		{200, "0.0520244459"},
		{300, "0.0523616267"},
		{364, "0.0527013471"}, // Excel would give ~0.0533938 here (~7 bp too high)
	}
	tol := mustDec("0.0000001")
	for _, c := range cases {
		mat := settle.AddDate(0, 0, c.dsm)
		got, err := TBillEq(settle, mat, mustDec("0.05"), Options{Scale: 12})
		if err != nil {
			t.Fatalf("dsm=%d: %v", c.dsm, err)
		}
		if !approxEqual(t, got, mustDec(c.want), tol) {
			t.Errorf("dsm=%d: got=%s want~%s", c.dsm, got.String(), c.want)
		}
	}
}
