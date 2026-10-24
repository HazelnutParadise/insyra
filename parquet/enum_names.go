package parquet

import "fmt"

// String returns the constant name of v, or Compression(n) for a value that
// is not one of the constants.
func (c Compression) String() string {
	switch c {
	case CompressionNone:
		return "CompressionNone"
	case CompressionSnappy:
		return "CompressionSnappy"
	case CompressionGzip:
		return "CompressionGzip"
	case CompressionBrotli:
		return "CompressionBrotli"
	case CompressionZstd:
		return "CompressionZstd"
	}
	return fmt.Sprintf("Compression(%d)", int(c))
}
