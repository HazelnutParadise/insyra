package nn

import "math"

// f16BitsToFloat32 widens an IEEE 754 binary16 value without changing its
// value. Every binary16 value is exactly representable as binary32.
func f16BitsToFloat32(bits uint16) float32 {
	sign := uint32(bits&0x8000) << 16
	exponent := int((bits >> 10) & 0x1f)
	mantissa := uint32(bits & 0x03ff)
	switch exponent {
	case 0:
		if mantissa == 0 {
			return math.Float32frombits(sign)
		}
		// Normalize the binary16 subnormal while retaining its exact
		// significand, then encode it as a binary32 normal.
		exponent = 1
		for mantissa&0x0400 == 0 {
			mantissa <<= 1
			exponent--
		}
		mantissa &= 0x03ff
		return math.Float32frombits(sign | uint32(exponent+112)<<23 | mantissa<<13)
	case 0x1f:
		return math.Float32frombits(sign | 0x7f800000 | mantissa<<13)
	default:
		return math.Float32frombits(sign | uint32(exponent+112)<<23 | mantissa<<13)
	}
}

// bf16BitsToFloat32 widens the top 16 bits of a binary32 value. This is a
// bit-preserving conversion, including signed zero, subnormals, infinities,
// and NaNs.
func bf16BitsToFloat32(bits uint16) float32 {
	return math.Float32frombits(uint32(bits) << 16)
}

// float32ToF16Bits converts binary32 to binary16 with round-to-nearest-even.
// It is used only for an ONNX Cast target; tensors continue to carry f32.
func float32ToF16Bits(value float32) uint16 {
	bits := math.Float32bits(value)
	sign := uint16(bits>>16) & 0x8000
	exponent := int((bits >> 23) & 0xff)
	mantissa := bits & 0x007fffff

	if exponent == 0xff {
		if mantissa == 0 {
			return sign | 0x7c00
		}
		result := uint16(mantissa >> 13)
		if result == 0 {
			result = 1
		}
		return sign | 0x7c00 | result
	}
	if exponent == 0 {
		return sign
	}

	unbiasedExponent := exponent - 127
	if unbiasedExponent > 15 {
		return sign | 0x7c00
	}
	if unbiasedExponent >= -14 {
		halfExponent := uint16(unbiasedExponent + 15)
		halfMantissa := uint16(mantissa >> 13)
		if shouldRoundToEven(mantissa&0x1fff, 0x1000, halfMantissa&1 != 0) {
			halfMantissa++
			if halfMantissa == 0x0400 {
				halfExponent++
				halfMantissa = 0
				if halfExponent == 0x1f {
					return sign | 0x7c00
				}
			}
		}
		return sign | halfExponent<<10 | halfMantissa
	}

	// Binary32 subnormals are far below the binary16 range. For binary32
	// normals in the range [-24, -15], round the hidden significand directly
	// into the binary16 subnormal field.
	if exponent < 103 {
		return sign
	}
	significand := mantissa | 0x00800000
	shift := uint(126 - exponent)
	halfMantissa := uint16(significand >> shift)
	remainderMask := uint32(1<<shift) - 1
	remainder := significand & remainderMask
	halfway := uint32(1) << (shift - 1)
	if shouldRoundToEven(remainder, halfway, halfMantissa&1 != 0) {
		halfMantissa++
	}
	return sign | halfMantissa
}

// float32ToBF16Bits converts binary32 to bfloat16 with round-to-nearest-even.
func float32ToBF16Bits(value float32) uint16 {
	bits := math.Float32bits(value)
	upper := uint16(bits >> 16)
	lower := bits & 0xffff
	if bits&0x7f800000 == 0x7f800000 && bits&0x007fffff != 0 {
		// Rounding a very small binary32 NaN could otherwise erase its payload
		// and turn it into infinity. Keep it a NaN while retaining its payload
		// whenever the top 16 bits already carry one.
		if upper&0x007f == 0 {
			upper |= 1
		}
		return upper
	}
	if shouldRoundToEven(lower, 0x8000, upper&1 != 0) {
		upper++
	}
	return upper
}

func shouldRoundToEven(remainder, halfway uint32, lowBitSet bool) bool {
	return remainder > halfway || remainder == halfway && lowBitSet
}
