#pragma once

#include <stdbool.h>
#include <stdint.h>

// Canonical quiet NaN values as defined by the WebAssembly spec:
// sign=0, exponent all-ones, MSB of mantissa=1 (quiet bit), payload=0.
//
// JIT (Cranelift/LLVM, canonicalize_nans=true) always produces these values
// when an operation yields NaN. WAVM executes float ops via cross-module calls
// into this soft-float library, so the library must return the same values to
// keep execution traces in sync and prevent fraud-proof divergence.
#define F32_CANONICAL_NAN UINT32_C(0x7FC00000)
#define F64_CANONICAL_NAN UINT64_C(0x7FF8000000000000)

// f32 NaN: exponent bits 30-23 all set, mantissa bits 22-0 non-zero.
static inline bool f32_bits_isNaN(uint32_t v) {
	return (v & 0x7F800000u) == 0x7F800000u && (v & 0x007FFFFFu) != 0;
}

// f64 NaN: exponent bits 62-52 all set, mantissa bits 51-0 non-zero.
static inline bool f64_bits_isNaN(uint64_t v) {
	return (v & UINT64_C(0x7FF0000000000000)) == UINT64_C(0x7FF0000000000000)
	    && (v & UINT64_C(0x000FFFFFFFFFFFFF)) != 0;
}

static inline uint32_t f32_canonicalize(uint32_t v) {
	return f32_bits_isNaN(v) ? F32_CANONICAL_NAN : v;
}

static inline uint64_t f64_canonicalize(uint64_t v) {
	return f64_bits_isNaN(v) ? F64_CANONICAL_NAN : v;
}
