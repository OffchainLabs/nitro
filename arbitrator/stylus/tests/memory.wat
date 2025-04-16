;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

(module
    (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
    (import "vm_hooks" "read_args"           (func $read_args    (param i32)))
    (import "vm_hooks" "write_result"        (func $write_result (param i32 i32)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
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
            local.get $size
            i32.gt_u
            (if (then
                    (i32.sub (local.get $size) (memory.size))
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
        call $write_result

        ;; return success
        i32.const 0
    )
    (memory (export "memory") 1 128)
)
