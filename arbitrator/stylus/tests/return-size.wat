;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
    (import "vm_hooks" "read_args"           (func $read_args    (param i32)))
    (import "vm_hooks" "write_result"        (func $write_result (param i32 i32)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (local $size i32)

		;; read input
        i32.const 0
        call $read_args

        ;; read the target size from the last 4 bytes of the input big endian
        ;; byte 1
		local.get $args_len
        i32.const 1
        i32.sub
        local.tee $size
		i32.load8_u

        ;; byte 2
        local.get $size
        i32.const 1
        i32.sub
        local.tee $size
		i32.load8_u
        i32.const 8
        i32.shl
        i32.or

        ;; byte 3
        local.get $size
        i32.const 1
        i32.sub
        local.tee $size
		i32.load8_u
        i32.const 16
        i32.shl
        i32.or

        ;; byte 4
        local.get $size
        i32.const 1
        i32.sub
        local.tee $size
		i32.load8_u
        i32.const 32
        i32.shl
        i32.or

        local.tee $size

		;; grow memory enough to handle the output
		;; we start with one page allocated, so no need to round up
		i32.const 65536
		i32.div_u
		memory.grow
        drop

        ;; set return data
        i32.const 0
		local.get $size
        call $write_result

        ;; return success
        i32.const 0
    )
    (memory (export "memory") 1)
)
