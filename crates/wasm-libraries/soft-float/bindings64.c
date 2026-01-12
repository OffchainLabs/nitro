#include "softfloat.h"

bool f64_isReal(float64_t f) {
	uint64_t exponentMask = (1ull << 63) - (1ull << 52);
	return (f.v & exponentMask) != exponentMask;
}

bool f64_isNaN(float64_t f) {
	if (f64_isReal(f)) return false;
	uint64_t fraction = f.v & ((1ull << 52) - 1);
	return fraction != 0;
}

bool f64_isInfinity(float64_t f) {
	if (f64_isReal(f)) return false;
	uint64_t fraction = f.v & ((1ull << 52) - 1);
	return fraction == 0;
}

const uint64_t F64_SIGN_BIT = 1ull << 63;

bool f64_isNegative(float64_t f) {
	return f.v & F64_SIGN_BIT;
}

bool f64_isZero(float64_t f) {
	return (f.v & ~F64_SIGN_BIT) == 0;
}

float64_t f64_positiveZero() {
	float64_t f = {0};
	return f;
}

float64_t f64_negativeZero() {
	float64_t f = {F64_SIGN_BIT};
	return f;
}

uint64_t wavm__f64_abs(uint64_t v) {
	v &= ~F64_SIGN_BIT;
	return v;
}

uint64_t wavm__f64_neg(uint64_t v) {
	v ^= F64_SIGN_BIT;
	return v;
}

uint64_t wavm__f64_ceil(uint64_t v) {
	float64_t f = {v};
	f = f64_roundToInt(f, softfloat_round_max, true);
	return f.v;
}

uint64_t wavm__f64_floor(uint64_t v) {
	float64_t f = {v};
	f = f64_roundToInt(f, softfloat_round_min, true);
	return f.v;
}

uint64_t wavm__f64_trunc(uint64_t v) {
	float64_t f = {v};
	f = f64_roundToInt(f, softfloat_round_minMag, true);
	return f.v;
}

uint64_t wavm__f64_nearest(uint64_t v) {
	float64_t f = {v};
	f = f64_roundToInt(f, softfloat_round_near_even, true);
	return f.v;
}

uint64_t wavm__f64_sqrt(uint64_t v) {
	float64_t f = {v};
	f = f64_sqrt(f);
	return f.v;
}

uint64_t wavm__f64_add(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	float64_t f = f64_add(a, b);
	return f.v;
}

uint64_t wavm__f64_sub(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	float64_t f = f64_sub(a, b);
	return f.v;
}

uint64_t wavm__f64_mul(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	float64_t f = f64_mul(a, b);
	return f.v;
}

uint64_t wavm__f64_div(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	float64_t f = f64_div(a, b);
	return f.v;
}

uint64_t wavm__f64_min(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	if (f64_isNaN(a)) {
		return a.v;
	} else if (f64_isNaN(b)) {
		return b.v;
	} else if (f64_isInfinity(a) && f64_isNegative(a)) {
		return a.v;
	} else if (f64_isInfinity(b) && f64_isNegative(b)) {
		return b.v;
	} else if (f64_isInfinity(a) && !f64_isNegative(a)) {
		return b.v;
	} else if (f64_isInfinity(b) && !f64_isNegative(b)) {
		return a.v;
	} else if (f64_isZero(a) && f64_isZero(b) && (f64_isNegative(a) != f64_isNegative(b))) {
		return f64_negativeZero().v;
	} else {
		if (f64_lt(b, a)) {
			return b.v;
		} else {
			return a.v;
		}
	}
}

uint64_t wavm__f64_max(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	if (f64_isNaN(a)) {
		return a.v;
	} else if (f64_isNaN(b)) {
		return b.v;
	} else if (f64_isInfinity(b) && !f64_isNegative(b)) {
		return b.v;
	} else if (f64_isInfinity(a) && f64_isNegative(a)) {
		return b.v;
	} else if (f64_isInfinity(b) && f64_isNegative(b)) {
		return a.v;
	} else if (f64_isZero(a) && f64_isZero(b) && (f64_isNegative(a) != f64_isNegative(b))) {
		return f64_positiveZero().v;
	} else {
		if (f64_lt(a, b)) {
			return b.v;
		} else {
			return a.v;
		}
	}
}

uint64_t wavm__f64_copysign(uint64_t va, uint64_t vb) {
	va &= ~F64_SIGN_BIT;
	va |= (vb & F64_SIGN_BIT);
	return va;
}

uint8_t wavm__f64_eq(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	return f64_eq(a, b);
}

uint8_t wavm__f64_ne(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	return !f64_eq(a, b);
}

uint8_t wavm__f64_lt(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	return f64_lt(a, b);
}

uint8_t wavm__f64_le(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	return f64_le(a, b);
}

uint8_t wavm__f64_gt(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	if (f64_isNaN(a) || f64_isNaN(b)) return false;
	return !f64_le(a, b);
}

uint8_t wavm__f64_ge(uint64_t va, uint64_t vb) {
	float64_t a = {va};
	float64_t b = {vb};
	if (f64_isNaN(a) || f64_isNaN(b)) return false;
	return !f64_lt(a, b);
}

int32_t wavm__i32_trunc_f64_s(uint64_t v) {
	// signed truncation is defined over (i32::min - 1, i32::max + 1)
	float64_t max = {0x41e0000000000000};  // i32::max + 1 = 0x41e0000000000000
	float64_t min = {0xc1e0000000200000};  // i32::min - 1 = 0xc1e0000000200000
	float64_t val = {v};
	if (f64_isNaN(val) || f64_le(max, val) || f64_le(val, min)) {
		__builtin_trap();
	}
	return f64_to_i32(val, softfloat_round_minMag, true);
}

int32_t wavm__i32_trunc_sat_f64_s(uint64_t v) {
	// signed truncation is defined over (i32::min - 1, i32::max + 1)
	float64_t max = {0x41e0000000000000};  // i32::max + 1 = 0x41e0000000000000
	float64_t min = {0xc1e0000000200000};  // i32::min - 1 = 0xc1e0000000200000
	float64_t val = {v};
	if (f64_isNaN(val)) {
		return 0;
	}
	if (f64_le(max, val)) {
		return 2147483647;
	}
	if (f64_le(val, min)) {
		return -2147483648;
	}
	return f64_to_i32(val, softfloat_round_minMag, true);
}

uint32_t wavm__i32_trunc_f64_u(uint64_t v) {
	// unsigned truncation is defined over (-1, u32::max + 1)
	float64_t max = {0x41f0000000000000};  // u32::max + 1 = 0x41f0000000000000
	float64_t min = {0xbff0000000000000};  // -1           = 0xbff0000000000000
	float64_t val = {v};
	if (f64_isNaN(val) || f64_le(max, val) || f64_le(val, min)) {
		__builtin_trap();
	}
	if (f64_isNegative(val)) {
		return 0;
	}
	return f64_to_ui32(val, softfloat_round_minMag, true);
}

uint32_t wavm__i32_trunc_sat_f64_u(uint64_t v) {
	// unsigned truncation is defined over (-1, u32::max + 1)
	float64_t max = {0x41f0000000000000};  // u32::max + 1 = 0x41f0000000000000
	float64_t val = {v};
	if (f64_isNaN(val) || f64_isNegative(val)) {
		return 0;
	}
	if (f64_le(max, val)) {
		return ~0u;
	}
	return f64_to_ui32(val, softfloat_round_minMag, true);
}

int64_t wavm__i64_trunc_f64_s(uint64_t v) {
	// signed truncation is defined over (i64::min - 1, u64::max + 1)
	float64_t max = {0x43e0000000000000};  // i64::max + 1 = 0x43e0000000000000
	float64_t min = {0xc3e0000000000001};  // i64::min - 1 = 0xc3e0000000000000 (adjusted due to rounding)
	float64_t val = {v};
	if (f64_isNaN(val) || f64_le(max, val) || f64_le(val, min)) {
		__builtin_trap();
	}
	return f64_to_i64(val, softfloat_round_minMag, true);
}

int64_t wavm__i64_trunc_sat_f64_s(uint64_t v) {
	// signed truncation is defined over (i64::min - 1, u64::max + 1)
	float64_t max = {0x43e0000000000000};  // i64::max + 1 = 0x43e0000000000000
	float64_t min = {0xc3e0000000000001};  // i64::min - 1 = 0xc3e0000000000000 (adjusted due to rounding)
	float64_t val = {v};
	if (f64_isNaN(val)) {
		return 0;
	}
	if (f64_le(max, val)) {
		return 9223372036854775807ll;
	}
	if (f64_le(val, min)) {
		return -(((int64_t) 1) << 63);
	}
	return f64_to_i64(val, softfloat_round_minMag, true);
}

uint64_t wavm__i64_trunc_f64_u(uint64_t v) {
	// unsigned truncation is defined over (-1, u64::max + 1)
	float64_t max = {0x43f0000000000000};  // u64::max + 1 = 0x43f0000000000000
	float64_t min = {0xbff0000000000000};  // -1           = 0xbff0000000000000
	float64_t f = {v};
	if (f64_isNaN(f) || f64_le(max, f) || f64_le(f, min)) {
		__builtin_trap();
	}
	if (f64_isNegative(f)) {
		return 0;
	}
	return f64_to_ui64(f, softfloat_round_minMag, true);
}

uint64_t wavm__i64_trunc_sat_f64_u(uint64_t v) {
	// unsigned truncation is defined over (-1, u64::max + 1)
	float64_t max = {0x43f0000000000000};  // u64::max + 1 = 0x43f0000000000000
	float64_t val = {v};
	if (f64_isNaN(val) || f64_isNegative(val)) {
		return 0;
	}
	if (f64_le(max, val)) {
		return 18446744073709551615ull;
	}
	return f64_to_ui64(val, softfloat_round_minMag, true);
}

uint64_t wavm__f64_convert_i32_s(int32_t x) {
	return i32_to_f64(x).v;
}

uint64_t wavm__f64_convert_i32_u(uint32_t x) {
	return ui32_to_f64(x).v;
}

uint64_t wavm__f64_convert_i64_s(int64_t x) {
	return i64_to_f64(x).v;
}

uint64_t wavm__f64_convert_i64_u(uint64_t x) {
	return ui64_to_f64(x).v;
}

uint32_t wavm__f32_demote_f64(uint64_t x) {
	float64_t f = {x};
	return f64_to_f32(f).v;
}

uint64_t wavm__f64_promote_f32(uint32_t x) {
	float32_t f = {x};
	return f32_to_f64(f).v;
}
