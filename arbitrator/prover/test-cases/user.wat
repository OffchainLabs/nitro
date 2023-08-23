;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (func $safe
        i32.const 5
        drop)
    (func $unreachable
        i32.const 0
        i64.const 4
        unreachable)
    (func $out_of_bounds
        i32.const 0xFFFFFF
        i32.load
        drop)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        local.get $args_len
        i32.const 1
        i32.eq
        (if
            (then (call $safe))
        )
        local.get $args_len
        i32.const 2
        i32.eq
        (if
            (then (call $unreachable))
        )
        local.get $args_len
        i32.const 3
        i32.eq
        (if
            (then (call $out_of_bounds))
        )
        i32.const 100)
    (memory 1 1))
