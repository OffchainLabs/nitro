;; Copyright 2026, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
;;
;; Tests that memory.fill only uses the low 8 bits of the value argument.
;; Calls memory.fill with value 0x100 (low 8 bits = 0x00), so all filled bytes must be 0x00.

(module
    (func (export "run")
        ;; fill 8 bytes at offset 0xaaa with value 0x100
        ;; only the low 8 bits (0x00) should be used, so memory must be all zeros
        (memory.fill (i32.const 0xaaa) (i32.const 0x100) (i32.const 8)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0))
    (memory (export "memory") 1 1))
