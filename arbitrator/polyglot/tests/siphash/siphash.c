// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
//
// Note for just this file: an MIT variant of this program may be found at
//     https://github.com/majek/csiphash/
//

#include <stdint.h>

// wasm is always little endian
#define _le64toh(x) ((uint64_t)(x))

#define ROTATE(x, b) (uint64_t)( ((x) << (b)) | ( (x) >> (64 - (b))) )

#define HALF_ROUND(a,b,c,d,s,t)			\
	a += b; c += d;				\
	b = ROTATE(b, s) ^ a;			\
	d = ROTATE(d, t) ^ c;			\
	a = ROTATE(a, 32);

#define DOUBLE_ROUND(v0,v1,v2,v3)		\
	HALF_ROUND(v0,v1,v2,v3,13,16);		\
	HALF_ROUND(v2,v1,v0,v3,17,21);		\
	HALF_ROUND(v0,v1,v2,v3,13,16);		\
	HALF_ROUND(v2,v1,v0,v3,17,21);


uint64_t siphash24(const void *src, unsigned long len, const uint8_t key[16]) {
	const uint64_t *_key = (uint64_t *)key;
	uint64_t k0 = _le64toh(_key[0]);
	uint64_t k1 = _le64toh(_key[1]);
	uint64_t b = (uint64_t)len << 56;
	const uint64_t *in = (uint64_t*)src;

	uint64_t v0 = k0 ^ 0x736f6d6570736575ULL;
	uint64_t v1 = k1 ^ 0x646f72616e646f6dULL;
	uint64_t v2 = k0 ^ 0x6c7967656e657261ULL;
	uint64_t v3 = k1 ^ 0x7465646279746573ULL;

	while (len >= 8) {
		uint64_t mi = _le64toh(*in);
		in += 1; len -= 8;
		v3 ^= mi;
		DOUBLE_ROUND(v0,v1,v2,v3);
		v0 ^= mi;
	}

	uint64_t t = 0; uint8_t *pt = (uint8_t *)&t; uint8_t *m = (uint8_t *)in;
	switch (len) {
	case 7: pt[6] = m[6];
	case 6: pt[5] = m[5];
	case 5: pt[4] = m[4];
	case 4: *((uint32_t*)&pt[0]) = *((uint32_t*)&m[0]); break;
	case 3: pt[2] = m[2];
	case 2: pt[1] = m[1];
	case 1: pt[0] = m[0];
	}
	b |= _le64toh(t);

	v3 ^= b;
	DOUBLE_ROUND(v0,v1,v2,v3);
	v0 ^= b; v2 ^= 0xff;
	DOUBLE_ROUND(v0,v1,v2,v3);
	DOUBLE_ROUND(v0,v1,v2,v3);
	return (v0 ^ v1) ^ (v2 ^ v3);
}
