;; Copyright 2026, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

(module
    (func (export "fill_overflow")
        ;; memory.fill(dst=0, val=0x100, len=1)
        ;; val > 0xFF should trigger MemoryFillValueOverflow trap
        (memory.fill (i32.const 0) (i32.const 0x100) (i32.const 1)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call 0)
        (i32.const 0)
    )
    (memory (export "memory") 1 1))
