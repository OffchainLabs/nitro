;; Copyright 2023-2024, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "console" "tee_i32" (func $tee_i32 (param i32) (result i32)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; fail to grow the memory a non-zero number of pages
        i32.const -65537
        call $tee_i32
        memory.grow
        call $tee_i32
        i32.const -1
        i32.eq
        i32.eqz
        (if (then unreachable))

        ;; succeed growing 0 pages
        i32.const 0
        memory.grow
        call $tee_i32
        i32.eqz
        i32.eqz
    )
    (memory (export "memory") 0 0)
)
