;; Copyright 2026, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

;; Reads page count N (u32, 4 bytes little-endian) from calldata and calls
;; memory.grow(N).  Returns 0 if grow returned a non-negative result (the old
;; page count, i.e. the grow succeeded), or 1 if grow returned -1 (the machine
;; ceiling was hit).  A non-zero return from user_entrypoint causes Stylus to
;; revert the transaction, so callers can distinguish success from ceiling
;; failure purely by receipt status.
(module
    (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
    (import "vm_hooks" "read_args"           (func $read_args    (param i32)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; Read the page count (4 bytes, little-endian u32) into memory[0..4]
        i32.const 0
        call $read_args

        ;; Load page count from memory[0] and grow
        i32.const 0
        i32.load
        memory.grow

        ;; Return 1 if grow returned -1 (failed), 0 if it returned a page count (succeeded)
        i32.const -1
        i32.eq
    )
    (memory (export "memory") 1)
)
