;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "forward" "memory_grow" (func (param i32)))
    (import "forward" "read_args"   (func $read_args   (param i32)))
    (import "forward" "return_data" (func $return_data (param i32 i32)))
    (func (export "arbitrum_main") (param $args_len i32) (result i32)
        (local $size i32) (local $step i32)

        ;; store the target size argument at offset 0
        i32.const 0
        call $read_args

        ;; copy the target size
        i32.const 0
        i32.load8_u
        local.set $size

        ;; copy the step size
        i32.const 1
        i32.load8_u
        local.set $step

        ;; grow until equal to the target size
        (loop $loop

            ;; grow by $step, shrinking if needed
            (i32.add (local.get $step) (memory.size))
            i32.const 128
            i32.gt_u
            (if (then
                    (i32.sub (i32.const 128) (memory.size))
                    local.set $step
            ))

            (memory.grow (local.get $step))
            drop

            ;; loop if too small
            (i32.lt_u (memory.size) (local.get $size))
            br_if $loop
        )

        ;; store the memory size at offset 0
        i32.const 0
        memory.size
        i32.store

        ;; make that single byte the return data
        i32.const 0
        i32.const 1
        call $return_data

        ;; return success
        i32.const 0
    )
    (memory (export "memory") 1 128)
)
