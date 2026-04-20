;; Copyright 2026, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

(module
    (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
    (import "vm_hooks" "read_args"           (func $read_args (param i32)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        i32.const 0
        call $read_args

        i32.const 0
        i32.load
        call $pay_for_memory_grow

        i32.const 0
    )
    (memory (export "memory") 1 128)
)
