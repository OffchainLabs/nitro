#include "softfloat.h"

uint32_t wavm__f32_abs(uint32_t v) {
	// Cut out the sign bit
	v &= ~(1 << 31);
	return v;
}

uint32_t wavm__f32_neg(uint32_t v) {
	// Flip the sign bit
	v ^= 1 << 31;
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
