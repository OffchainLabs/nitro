;; Copyright 2025, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md


(module
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1 1)
    (func $main (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; load args to 0x0
        (call $read_args (i32.const 0))

        ;; write result from args
        (call $write_result (i32.const 0) (local.get $args_len))

        ;; return success
        i32.const 0
    )
)


