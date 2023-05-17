;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (func (export "fill")
        (memory.fill (i32.const 0xffff) (i32.const 0) (i32.const 2)))
    (func (export "copy_left")
        (memory.copy (i32.const 0xffff) (i32.const 0xfffe) (i32.const 2)))
    (func (export "copy_right")
        (memory.copy (i32.const 0xfffe) (i32.const 0xffff) (i32.const 2)))
    (func (export "copy_same")
        (memory.copy (i32.const 0xffff) (i32.const 0xffff) (i32.const 2)))
    (data (i32.const 0xfffe) "\01\02") ;; last two bytes shouldn't change
    (memory (export "memory") 1 1))
