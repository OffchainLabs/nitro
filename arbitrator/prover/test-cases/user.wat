;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "vm_hooks" "storage_load_bytes32" (func $storage_load_bytes32 (param i32 i32)))

    (func $storage_load (result i32)
        i32.const 0
        i32.const 32
        call $storage_load_bytes32
        i32.const 0
    )
    (func $safe (result i32)
        i32.const 5
    )
    (func $unreachable (result i32)
        i32.const 0
        i64.const 4
        unreachable
    )
    (func $out_of_bounds (result i32)
        i32.const 0xFFFFFF
        i32.load
    )
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; this func uses $args_len to select which func to call

        ;; only call that succeeds
        (i32.eq (local.get $args_len) (i32.const 1))
        (if
            (then (call $safe) (return))
        )

        ;; reverts due to an unreachable
        (i32.eq (local.get $args_len) (i32.const 2))
        (if
            (then (call $unreachable) (return))
        )

        ;; reverts due to an out of bounds memory access
        (i32.eq (local.get $args_len) (i32.const 3))
        (if
            (then (call $out_of_bounds) (return))
        )

        (i32.eq (local.get $args_len) (i32.const 32))
        (if
            (then (call $storage_load) (return))
        )

        unreachable
    )
    (memory (export "memory") 1 1))
