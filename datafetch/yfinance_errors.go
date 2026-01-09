// file: insyra/datafetch/yfinance_errors.go
package datafetch

import (
	"errors"

	yfclient "github.com/wnjoon/go-yfinance/pkg/client"
)

var (
	ErrRateLimited   = errors.New("yfinance: rate limited")
	ErrTimeout       = errors.New("yfinance: timeout")
	ErrInvalidSymbol = errors.New("yfinance: invalid symbol")
)

// classifyError maps go-yfinance client errors into stable errors you can check.
func classifyError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case yfclient.IsRateLimitError(err):
		return ErrRateLimited
	case yfclient.IsTimeoutError(err):
		return ErrTimeout
	case yfclient.IsInvalidSymbolError(err):
		return ErrInvalidSymbol
	default:
		return err
	}
}

// retryable returns true if the error is worth retrying.
func retryable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrRateLimited) || errors.Is(err, ErrTimeout)
}
