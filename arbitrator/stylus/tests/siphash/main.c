// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
//
// You can compile this file with stock clang as follows
//     clang *.c -o siphash.wasm --target=wasm32 --no-standard-libraries -mbulk-memory -Wl,--no-entry -Oz
//
// For C programs reliant on the standard library, cross compile clang with wasi
//     https://github.com/WebAssembly/wasi-sdk

#include "../../../langs/c/arbitrum.h"

extern uint64_t siphash24(const void *src, unsigned long len, const uint8_t key[16]);

ArbResult user_main(const uint8_t * args, size_t args_len) {
    const uint64_t hash = *(uint64_t *) args;
    const uint8_t * key = args + 8;
    const uint8_t * plaintext = args + 24;
    const uint64_t length = args_len - 24;

    uint8_t valid = siphash24(plaintext, length, key) == hash ? 0 : 1;

    return (ArbResult) {
        .status = valid,
        .output = args,
        .output_len = args_len,
    };
}

ARBITRUM_MAIN(user_main);
