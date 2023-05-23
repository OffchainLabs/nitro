;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "forward" "memory_grow" (func (param i32) (result)))
    (func (export "grow") (result i32)
        (local $i i32)
        (loop $loop
            (memory.grow (i32.const 1))
            i32.const 127
            i32.ne
            br_if $loop
        )
        memory.size
    )
    (memory (export "memory") 0 128)
)
