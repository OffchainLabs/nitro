;; Copyright 2026, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

;; Tests that memory.grow(1<<16) fails when the heap bound is below 1<<16.
;; The entrypoint returns 0 (success) iff memory.grow returned -1 (grow failed as expected).
(module
    (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        i32.const 65536
        memory.grow
        i32.const -1
        i32.eq
        i32.eqz
    )
    (memory (export "memory") 0)
)
