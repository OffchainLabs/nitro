;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (import "vm_hooks" "msg_value"    (func $test         (param i32)))
    (memory (export "memory") 1 1)
    (func $main (export "arbitrum_main") (param $args_len i32) (result i32)
        (local $i i32)

        ;; write args to 0x0
        i32.const 0
        call $read_args

        ;; treat first 4 bytes as # of iterations
        (i32.load (i32.const 0))
        local.set $i

        (loop
            ;; call the test function
            i32.const 0
            call $test

            ;; decrement and loop
            (i32.sub (local.get $i) (i32.const 1))
            local.tee $i
            i32.const 0
            i32.ne
            br_if 0
        )

        ;; return success
        i32.const 0
    )
)
