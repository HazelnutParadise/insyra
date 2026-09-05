package quant

import (
	"math"
	"math/rand"
	"strings"
	"testing"
)

func TestBlackScholesHullExample(t *testing.T) {
	in := BSInput{Spot: 42, Strike: 40, Rate: 0.10, Volatility: 0.20, TimeToExpiry: 0.5}

	call, err := BlackScholes(in)
	if err != nil {
		t.Fatalf("call returned unexpected error: %v", err)
	}
	assertClose(t, call.Price, 4.7594, 1e-3)

	in.Type = OptionPut
	put, err := BlackScholes(in)
	if err != nil {
		t.Fatalf("put returned unexpected error: %v", err)
	}
	assertClose(t, put.Price, 0.8086, 1e-3)
}

func TestBlackScholesDividendGolden(t *testing.T) {
	in := BSInput{
		Spot: 100, Strike: 100, Rate: 0.05, DividendYield: 0.02,
		Volatility: 0.25, TimeToExpiry: 1,
	}
	call, err := BlackScholes(in)
	if err != nil {
		t.Fatalf("call returned unexpected error: %v", err)
	}
	// Hand calculation: d1 = 0.245, d2 = -0.005,
	// N(d1) = 0.5967717843, N(d2) = 0.4980052969,
	// S*exp(-qT) = 98.01986733, K*exp(-rT) = 95.12294245.
	// Therefore call = 98.01986733*N(d1) - 95.12294245*N(d2)
	// = 11.1237619281. Put-call parity gives put = 8.2268370475.
	assertClose(t, call.Price, 11.1237619281, 1e-9)

	in.Type = OptionPut
	put, err := BlackScholes(in)
	if err != nil {
		t.Fatalf("put returned unexpected error: %v", err)
	}
	assertClose(t, put.Price, 8.2268370475, 1e-9)
}

func TestBlackScholesPutCallParity(t *testing.T) {
	rng := rand.New(rand.NewSource(20260904))
	for i := 0; i < 32; i++ {
		in := BSInput{
			Spot:          50 + 150*rng.Float64(),
			Strike:        50 + 150*rng.Float64(),
			Rate:          -0.02 + 0.12*rng.Float64(),
			DividendYield: 0.04 * rng.Float64(),
			Volatility:    0.05 + 0.75*rng.Float64(),
			TimeToExpiry:  0.05 + 4*rng.Float64(),
		}
		call, err := BlackScholes(in)
		if err != nil {
			t.Fatalf("case %d call returned unexpected error: %v", i, err)
		}
		in.Type = OptionPut
		put, err := BlackScholes(in)
		if err != nil {
			t.Fatalf("case %d put returned unexpected error: %v", i, err)
		}
		want := in.Spot*math.Exp(-in.DividendYield*in.TimeToExpiry) -
			in.Strike*math.Exp(-in.Rate*in.TimeToExpiry)
		assertClose(t, call.Price-put.Price, want, 1e-10)
	}
}

func TestBlackScholesGreeksFiniteDifferences(t *testing.T) {
	const h = 1e-5
	base := BSInput{
		Spot: 1, Strike: 1, Rate: 0.04, DividendYield: 0.015,
		Volatility: 0.30, TimeToExpiry: 1.2,
	}
	for _, optionType := range []OptionType{OptionCall, OptionPut} {
		base.Type = optionType
		got, err := BlackScholes(base)
		if err != nil {
			t.Fatalf("%v returned unexpected error: %v", optionType, err)
		}

		up := base
		up.Spot += h
		down := base
		down.Spot -= h
		upPrice := mustOptionPrice(t, up)
		downPrice := mustOptionPrice(t, down)
		assertClose(t, got.Delta, (upPrice-downPrice)/(2*h), 1e-6)
		assertClose(t, got.Gamma, (upPrice-2*got.Price+downPrice)/(h*h), 1e-4)

		up = base
		up.Volatility += h
		down = base
		down.Volatility -= h
		assertClose(t, got.Vega, (mustOptionPrice(t, up)-mustOptionPrice(t, down))/(2*h), 1e-6)

		up = base
		up.TimeToExpiry += h
		down = base
		down.TimeToExpiry -= h
		assertClose(t, got.Theta, -(mustOptionPrice(t, up)-mustOptionPrice(t, down))/(2*h), 1e-6)

		up = base
		up.Rate += h
		down = base
		down.Rate -= h
		assertClose(t, got.Rho, (mustOptionPrice(t, up)-mustOptionPrice(t, down))/(2*h), 1e-6)
	}
}

func TestBlackScholesExpiry(t *testing.T) {
	call, err := BlackScholes(BSInput{Spot: 105, Strike: 100, Volatility: 0.2, TimeToExpiry: 0, Type: OptionCall})
	if err != nil {
		t.Fatalf("call returned unexpected error: %v", err)
	}
	assertClose(t, call.Price, 5, 0)
	assertClose(t, call.Delta, 1, 0)
	assertClose(t, call.Gamma, 0, 0)
	assertClose(t, call.Vega, 0, 0)
	assertClose(t, call.Theta, 0, 0)
	assertClose(t, call.Rho, 0, 0)

	put, err := BlackScholes(BSInput{Spot: 105, Strike: 100, Volatility: 0.2, TimeToExpiry: 0, Type: OptionPut})
	if err != nil {
		t.Fatalf("put returned unexpected error: %v", err)
	}
	assertClose(t, put.Price, 0, 0)
	assertClose(t, put.Delta, 0, 0)
	assertClose(t, put.Gamma, 0, 0)
	assertClose(t, put.Vega, 0, 0)
	assertClose(t, put.Theta, 0, 0)
	assertClose(t, put.Rho, 0, 0)
}

func TestBlackScholesRejectsInvalidInput(t *testing.T) {
	valid := BSInput{Spot: 100, Strike: 100, Rate: 0.05, DividendYield: 0.01, Volatility: 0.2, TimeToExpiry: 1, Type: OptionCall}
	cases := []struct {
		name  string
		input BSInput
		field string
	}{
		{"spot", func() BSInput { in := valid; in.Spot = 0; return in }(), "Spot"},
		{"strike", func() BSInput { in := valid; in.Strike = -1; return in }(), "Strike"},
		{"volatility", func() BSInput { in := valid; in.Volatility = 0; return in }(), "Volatility"},
		{"negative expiry", func() BSInput { in := valid; in.TimeToExpiry = -0.1; return in }(), "TimeToExpiry"},
		{"nan spot", func() BSInput { in := valid; in.Spot = math.NaN(); return in }(), "Spot"},
		{"infinite strike", func() BSInput { in := valid; in.Strike = math.Inf(1); return in }(), "Strike"},
		{"nan rate", func() BSInput { in := valid; in.Rate = math.NaN(); return in }(), "Rate"},
		{"infinite dividend yield", func() BSInput { in := valid; in.DividendYield = math.Inf(-1); return in }(), "DividendYield"},
		{"unknown type", func() BSInput { in := valid; in.Type = OptionType(99); return in }(), "Type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BlackScholes(tc.input); err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Errorf("error = %v, want mention %q", err, tc.field)
			}
		})
	}
}

func TestImpliedVolatilityRoundTrips(t *testing.T) {
	cases := []struct {
		name       string
		spot       float64
		strike     float64
		optionType OptionType
	}{
		{"call ITM", 110, 100, OptionCall},
		{"call OTM", 90, 100, OptionCall},
		{"put ITM", 90, 100, OptionPut},
		{"put OTM", 110, 100, OptionPut},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := BSInput{Spot: tc.spot, Strike: tc.strike, Rate: 0.03, DividendYield: 0.01, Volatility: 0.35, TimeToExpiry: 0.75, Type: tc.optionType}
			priced, err := BlackScholes(in)
			if err != nil {
				t.Fatalf("BlackScholes returned unexpected error: %v", err)
			}
			in.Volatility = 0 // ImpliedVolatility must ignore this field.
			got, err := ImpliedVolatility(priced.Price, in)
			if err != nil {
				t.Fatalf("ImpliedVolatility returned unexpected error: %v", err)
			}
			assertClose(t, got, 0.35, 1e-8)
		})
	}
}

func TestImpliedVolatilityDeepOutOfTheMoney(t *testing.T) {
	in := BSInput{Spot: 100, Strike: 150, Rate: 0.02, Volatility: 0.6, TimeToExpiry: 0.25, Type: OptionCall}
	priced, err := BlackScholes(in)
	if err != nil {
		t.Fatalf("BlackScholes returned unexpected error: %v", err)
	}
	got, err := ImpliedVolatility(priced.Price, in)
	if err != nil {
		t.Fatalf("ImpliedVolatility returned unexpected error: %v", err)
	}
	assertClose(t, got, 0.6, 1e-6)
}

func TestImpliedVolatilityRejectsPricesOutsideBounds(t *testing.T) {
	in := BSInput{Spot: 100, Strike: 100, Rate: 0.05, Volatility: 0.2, TimeToExpiry: 1, Type: OptionCall}
	cases := []struct {
		name  string
		price float64
		want  string
	}{
		{"above upper bound", 101, "upper bound"},
		{"below lower bound", -0.01, "lower bound"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ImpliedVolatility(tc.price, in); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want mention %q", err, tc.want)
			}
		})
	}

	in.Type = OptionPut
	putUpper := in.Strike * math.Exp(-in.Rate*in.TimeToExpiry)
	if _, err := ImpliedVolatility(putUpper+1, in); err == nil || !strings.Contains(err.Error(), "upper bound") {
		t.Errorf("put upper-bound error = %v, want upper bound", err)
	}
}

func TestImpliedVolatilityRejectsExpiry(t *testing.T) {
	in := BSInput{Spot: 105, Strike: 100, Volatility: 0.2, TimeToExpiry: 0, Type: OptionCall}
	if _, err := ImpliedVolatility(5, in); err == nil || !strings.Contains(err.Error(), "TimeToExpiry") {
		t.Errorf("error = %v, want mention TimeToExpiry", err)
	}
}

func mustOptionPrice(t *testing.T, in BSInput) float64 {
	t.Helper()
	result, err := BlackScholes(in)
	if err != nil {
		t.Fatalf("BlackScholes returned unexpected error: %v", err)
	}
	return result.Price
}
