;; Copyright 2024, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (import "vm_hooks" "exit_early"   (func $exit         (param i32)))
    (memory (export "memory") 1 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; write args to offset 0
        (call $read_args (i32.const 0))

        ;; set the args as the result
        (call $write_result (i32.const 0) (local.get $args_len))

        ;; perform a hard revert (results should be discarded)
        unreachable
    )
)
