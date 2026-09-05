package quant

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra/stats"
)

// OptionType identifies whether an option is a call or a put.
type OptionType uint8

const (
	// OptionCall is a call option.
	OptionCall OptionType = iota
	// OptionPut is a put option.
	OptionPut
)

// BSInput contains the inputs to the Black-Scholes-Merton model. Rate and
// DividendYield are continuously compounded annual rates expressed as
// decimals, Volatility is annualized, and TimeToExpiry is measured in years.
type BSInput struct {
	Spot          float64
	Strike        float64
	Rate          float64
	DividendYield float64
	Volatility    float64
	TimeToExpiry  float64
	Type          OptionType
}

// BSResult contains the Black-Scholes-Merton option price and greeks. Vega is
// the derivative per unit of volatility, not per 1%; divide it by 100 for a
// one-percentage-point move. Theta is the negative derivative per year, not
// per day; divide it by 365 for a one-day figure.
type BSResult struct {
	Price float64
	Delta float64
	Gamma float64
	Vega  float64
	Theta float64
	Rho   float64
}

// BlackScholes prices a European call or put and returns its five greeks using
// the Black-Scholes-Merton model with a continuously compounded dividend yield.
// Rates and volatility are decimals, TimeToExpiry is in years, Vega is per
// unit of volatility, and Theta is per year. At zero time to expiry, Price is
// intrinsic value and the greeks use their limiting values.
//
// It returns an error if a required input is non-finite, Spot, Strike, or
// Volatility is not positive, TimeToExpiry is negative, or Type is unknown.
func BlackScholes(in BSInput) (*BSResult, error) {
	if err := validateBSInput(in); err != nil {
		return nil, err
	}
	return blackScholesF64(in)
}

func validateBSInput(in BSInput) error {
	inputs := []struct {
		name  string
		value float64
	}{
		{"Spot", in.Spot},
		{"Strike", in.Strike},
		{"Rate", in.Rate},
		{"DividendYield", in.DividendYield},
		{"Volatility", in.Volatility},
		{"TimeToExpiry", in.TimeToExpiry},
	}
	for _, input := range inputs {
		if !isFinite(input.value) {
			return fmt.Errorf("BlackScholes: %s must be finite, got %v", input.name, input.value)
		}
	}
	if in.Spot <= 0 {
		return fmt.Errorf("BlackScholes: Spot must be positive, got %v", in.Spot)
	}
	if in.Strike <= 0 {
		return fmt.Errorf("BlackScholes: Strike must be positive, got %v", in.Strike)
	}
	if in.Volatility <= 0 {
		return fmt.Errorf("BlackScholes: Volatility must be positive, got %v", in.Volatility)
	}
	if in.TimeToExpiry < 0 {
		return fmt.Errorf("BlackScholes: TimeToExpiry must be non-negative, got %v", in.TimeToExpiry)
	}
	if in.Type != OptionCall && in.Type != OptionPut {
		return fmt.Errorf("BlackScholes: unknown Type %d", in.Type)
	}
	return nil
}

func blackScholesF64(in BSInput) (*BSResult, error) {
	discountSpot := math.Exp(-in.DividendYield * in.TimeToExpiry)
	discountStrike := math.Exp(-in.Rate * in.TimeToExpiry)
	if in.TimeToExpiry == 0 {
		result := &BSResult{}
		switch in.Type {
		case OptionCall:
			if in.Spot > in.Strike {
				result.Price = in.Spot - in.Strike
				result.Delta = 1
			}
		case OptionPut:
			if in.Strike > in.Spot {
				result.Price = in.Strike - in.Spot
				result.Delta = -1
			}
		}
		return result, nil
	}

	sqrtTime := math.Sqrt(in.TimeToExpiry)
	volatilityTime := in.Volatility * sqrtTime
	d1 := (math.Log(in.Spot/in.Strike) +
		(in.Rate-in.DividendYield+0.5*in.Volatility*in.Volatility)*in.TimeToExpiry) / volatilityTime
	d2 := d1 - volatilityTime
	nd1 := stats.NormCDF(d1)
	nd2 := stats.NormCDF(d2)
	notND1 := stats.NormCDF(-d1)
	notND2 := stats.NormCDF(-d2)
	density := normPDF(d1)

	result := &BSResult{
		Gamma: discountSpot * density / (in.Spot * volatilityTime),
		Vega:  in.Spot * discountSpot * density * sqrtTime,
	}
	switch in.Type {
	case OptionCall:
		result.Price = in.Spot*discountSpot*nd1 - in.Strike*discountStrike*nd2
		result.Delta = discountSpot * nd1
		result.Theta = -in.Spot*discountSpot*density*in.Volatility/(2*sqrtTime) -
			in.Rate*in.Strike*discountStrike*nd2 + in.DividendYield*in.Spot*discountSpot*nd1
		result.Rho = in.Strike * in.TimeToExpiry * discountStrike * nd2
	case OptionPut:
		result.Price = in.Strike*discountStrike*notND2 - in.Spot*discountSpot*notND1
		result.Delta = -discountSpot * notND1
		result.Theta = -in.Spot*discountSpot*density*in.Volatility/(2*sqrtTime) +
			in.Rate*in.Strike*discountStrike*notND2 - in.DividendYield*in.Spot*discountSpot*notND1
		result.Rho = -in.Strike * in.TimeToExpiry * discountStrike * notND2
	}
	return result, nil
}

// ImpliedVolatility returns the volatility that makes BlackScholes(in) have
// the supplied price. The Volatility field in in is ignored. The solver first
// brackets the solution on [1e-6, 10] by bisection and then uses Newton
// polishing with vega as the derivative. It stops at a price error of 1e-10 or
// a volatility change of 1e-12, and never performs more than 200 iterations.
//
// It returns an error if price is non-finite, TimeToExpiry is zero, the price
// is outside the option's no-arbitrage lower or upper bound, or the capped
// iteration budget cannot find a solution.
func ImpliedVolatility(price float64, in BSInput) (float64, error) {
	if !isFinite(price) {
		return math.NaN(), fmt.Errorf("ImpliedVolatility: price must be finite, got %v", price)
	}
	validationInput := in
	validationInput.Volatility = 1
	if err := validateBSInput(validationInput); err != nil {
		return math.NaN(), err
	}
	if in.TimeToExpiry == 0 {
		return math.NaN(), fmt.Errorf("ImpliedVolatility: TimeToExpiry must be positive")
	}
	return impliedVolatilityF64(price, in)
}

func impliedVolatilityF64(price float64, in BSInput) (float64, error) {
	const (
		minVol         = 1e-6
		maxVol         = 10.0
		priceTolerance = 1e-10
		volTolerance   = 1e-12
		maxIterations  = 200
		bisectionLimit = 100
	)

	discountSpot := math.Exp(-in.DividendYield * in.TimeToExpiry)
	discountStrike := math.Exp(-in.Rate * in.TimeToExpiry)
	var lowerBound, upperBound float64
	switch in.Type {
	case OptionCall:
		lowerBound = math.Max(discountSpot*in.Spot-discountStrike*in.Strike, 0)
		upperBound = discountSpot * in.Spot
	case OptionPut:
		lowerBound = math.Max(discountStrike*in.Strike-discountSpot*in.Spot, 0)
		upperBound = discountStrike * in.Strike
	}
	if price < lowerBound {
		return math.NaN(), fmt.Errorf("ImpliedVolatility: price %.17g is below the lower bound %.17g", price, lowerBound)
	}
	if price > upperBound {
		return math.NaN(), fmt.Errorf("ImpliedVolatility: price %.17g is above the upper bound %.17g", price, upperBound)
	}

	low, high := minVol, maxVol
	lowInput, highInput := in, in
	lowInput.Volatility = low
	highInput.Volatility = high
	lowPrice, err := blackScholesF64(lowInput)
	if err != nil {
		return math.NaN(), err
	}
	highPrice, err := blackScholesF64(highInput)
	if err != nil {
		return math.NaN(), err
	}
	if price < lowPrice.Price || price > highPrice.Price {
		return math.NaN(), fmt.Errorf("ImpliedVolatility: price is outside the solvable volatility bracket [%.0e, %.0f]", minVol, maxVol)
	}

	vol := (low + high) / 2
	iterations := 0
	for ; iterations < bisectionLimit && iterations < maxIterations; iterations++ {
		mid := (low + high) / 2
		midInput := in
		midInput.Volatility = mid
		result, err := blackScholesF64(midInput)
		if err != nil {
			return math.NaN(), err
		}
		vol = mid
		priceError := result.Price - price
		if math.Abs(priceError) <= priceTolerance {
			return vol, nil
		}
		if priceError > 0 {
			high = mid
		} else {
			low = mid
		}
	}

	for ; iterations < maxIterations; iterations++ {
		currentInput := in
		currentInput.Volatility = vol
		result, err := blackScholesF64(currentInput)
		if err != nil {
			return math.NaN(), err
		}
		priceError := result.Price - price
		if math.Abs(priceError) <= priceTolerance {
			return vol, nil
		}
		if priceError > 0 {
			high = vol
		} else {
			low = vol
		}
		if result.Vega <= 0 || !isFinite(result.Vega) {
			break
		}
		next := vol - priceError/result.Vega
		if !isFinite(next) || next <= low || next >= high {
			next = (low + high) / 2
		}
		if math.Abs(next-vol) <= volTolerance {
			return next, nil
		}
		vol = next
	}

	return math.NaN(), fmt.Errorf("ImpliedVolatility: iteration limit of %d reached", maxIterations)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
