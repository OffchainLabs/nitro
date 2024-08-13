;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
    (import "vm_hooks" "read_args"           (func $read_args    (param i32)))
    (import "vm_hooks" "write_result"        (func $write_result (param i32 i32)))
    (import "console" "tee_i32"              (func $tee_i32 (param i32) (result i32)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        local.get $args_len
        i32.eqz
        (if (then
            ;; write an empty result to offset 0
            (call $write_result (i32.const 0) (i32.const 0))
            (return (i32.const 0))
        ))

        ;; grow 1 page so that we can read our args
        i32.const 1
        memory.grow
        drop

        ;; store the size argument at offset 0
        i32.const 0
        call $read_args

        ;; read the argument and grow the remainder
        i32.const 0
        i32.load8_u
        i32.const 1
        i32.sub
        memory.grow
        drop

        ;; write a result (should panic if out of bounds)
        i32.const 1
        i32.load
        i32.const 5
        i32.load
        call $write_result

        i32.const 0
    )
    (memory (export "memory") 0)
)
