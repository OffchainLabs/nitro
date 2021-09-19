#include "softfloat.h"

bool f32_isReal(float32_t f) {
	uint32_t exponentMask = (1u << 31) - (1u << 23);
	return (f.v & exponentMask) != exponentMask;
}

bool f32_isNaN(float32_t f) {
	if (f32_isReal(f)) return false;
	uint32_t fraction = f.v & ((1 << 23) - 1);
	return fraction != 0;
}

bool f32_isInfinity(float32_t f) {
	if (f32_isReal(f)) return false;
	uint32_t fraction = f.v & ((1 << 23) - 1);
	return fraction == 0;
}

const uint32_t SIGN_BIT = 1 << 31;

bool f32_isNegative(float32_t f) {
	return f.v & SIGN_BIT;
}

bool f32_isZero(float32_t f) {
	return (f.v & ~SIGN_BIT) == 0;
}

float32_t f32_positiveZero() {
	float32_t f = {0};
	return f;
}

float32_t f32_negativeZero() {
	float32_t f = {SIGN_BIT};
	return f;
}

uint32_t wavm__f32_abs(uint32_t v) {
	v &= ~SIGN_BIT;
	return v;
}

uint32_t wavm__f32_neg(uint32_t v) {
	v ^= SIGN_BIT;
	return v;
}

uint32_t wavm__f32_ceil(uint32_t v) {
	float32_t f = {v};
	f = f32_roundToInt(f, softfloat_round_max, true);
	return f.v;
}

uint32_t wavm__f32_floor(uint32_t v) {
	float32_t f = {v};
	f = f32_roundToInt(f, softfloat_round_min, true);
	return f.v;
}

uint32_t wavm__f32_trunc(uint32_t v) {
	float32_t f = {v};
	f = f32_roundToInt(f, softfloat_round_minMag, true);
	return f.v;
}

uint32_t wavm__f32_nearest(uint32_t v) {
	float32_t f = {v};
	f = f32_roundToInt(f, softfloat_round_near_even, true);
	return f.v;
}

uint32_t wavm__f32_sqrt(uint32_t v) {
	float32_t f = {v};
	f = f32_sqrt(f);
	return f.v;
}

uint32_t wavm__f32_add(uint32_t va, uint32_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	float32_t f = f32_add(a, b);
	return f.v;
}

uint32_t wavm__f32_sub(uint32_t va, uint32_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	float32_t f = f32_sub(a, b);
	return f.v;
}

uint32_t wavm__f32_mul(uint32_t va, uint32_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	float32_t f = f32_mul(a, b);
	return f.v;
}

uint32_t wavm__f32_div(uint32_t va, uint32_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	float32_t f = f32_div(a, b);
	return f.v;
}

uint32_t wavm__f32_min(uint32_t va, uint32_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	if (f32_isNaN(a) || f32_isNaN(b)) {
		return a.v;
	} else if (f32_isInfinity(a) && f32_isNegative(a)) {
		return a.v;
	} else if (f32_isInfinity(b) && f32_isNegative(b)) {
		return b.v;
	} else if (f32_isInfinity(a) && !f32_isNegative(a)) {
		return b.v;
	} else if (f32_isInfinity(b) && !f32_isNegative(b)) {
		return a.v;
	} else if (f32_isZero(a) && f32_isZero(b) && (f32_isNegative(a) != f32_isNegative(b))) {
		return f32_negativeZero().v;
	} else {
		if (f32_lt(b, a)) {
			return b.v;
		} else {
			return a.v;
		}
	}
}

uint32_t wavm__f32_max(uint32_t va, uint32_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	if (f32_isNaN(a) || f32_isNaN(b)) {
		return a.v;
	} else if (f32_isInfinity(a) && !f32_isNegative(a)) {
		return a.v;
	} else if (f32_isInfinity(b) && !f32_isNegative(b)) {
		return b.v;
	} else if (f32_isInfinity(a) && f32_isNegative(a)) {
		return b.v;
	} else if (f32_isInfinity(b) && f32_isNegative(b)) {
		return a.v;
	} else if (f32_isZero(a) && f32_isZero(b) && (f32_isNegative(a) != f32_isNegative(b))) {
		return f32_positiveZero().v;
	} else {
		if (f32_lt(a, b)) {
			return b.v;
		} else {
			return a.v;
		}
	}
}

uint32_t wavm__f32_copysign(uint32_t va, uint32_t vb) {
	va &= ~SIGN_BIT;
	va |= (vb & SIGN_BIT);
	return va;
}

uint8_t wavm__f32_eq(uint8_t va, uint8_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	return f32_eq(a, b);
}

uint8_t wavm__f32_ne(uint8_t va, uint8_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	if (f32_isNaN(a) || f32_isNaN(b)) return false;
	return !f32_eq(a, b);
}

uint8_t wavm__f32_lt(uint8_t va, uint8_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	return f32_lt(a, b);
}

uint8_t wavm__f32_le(uint8_t va, uint8_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	return f32_le(a, b);
}

uint8_t wavm__f32_gt(uint8_t va, uint8_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	if (f32_isNaN(a) || f32_isNaN(b)) return false;
	return !f32_le(a, b);
}

uint8_t wavm__f32_ge(uint8_t va, uint8_t vb) {
	float32_t a = {va};
	float32_t b = {vb};
	if (f32_isNaN(a) || f32_isNaN(b)) return false;
	return !f32_lt(a, b);
}
