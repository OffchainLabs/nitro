// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#ifndef ARBITRUM_HEADER_GUARD
#define ARBITRUM_HEADER_GUARD

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define USER_HOST import_module("vm_hooks")

extern __attribute__((USER_HOST, import_name("read_args"))) void read_args(const uint8_t * data);
extern __attribute__((USER_HOST, import_name("return_data"))) void return_data(const uint8_t * data, size_t len);
extern __attribute__((USER_HOST, import_name("memory_grow"))) void memory_grow(uint32_t pages);

typedef enum ArbStatus {
    Success = 0,
    Failure,
} ArbStatus;

typedef struct ArbResult {
    const ArbStatus status;
    const uint8_t * output;
    const size_t output_len;
} ArbResult;

#define ARBITRUM_MAIN(user_main)                                        \
    /* Force the compiler to import these symbols                    */ \
    /* Note: calling these functions will unproductively consume gas */ \
    __attribute__((export_name("mark_used")))                           \
    void mark_used() {                                                  \
        memory_grow(0);                                                 \
    }                                                                   \
                                                                        \
    __attribute__((export_name("arbitrum_main")))                       \
    int arbitrum_main(int args_len) {                                   \
        const uint8_t args[args_len];                                   \
        read_args(args);                                                \
        const ArbResult result = user_main(args, args_len);             \
        return_data(result.output, result.output_len);                  \
        return result.status;                                           \
    }

#ifdef __cplusplus
}
#endif

#endif
