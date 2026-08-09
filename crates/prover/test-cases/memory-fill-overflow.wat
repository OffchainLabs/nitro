;; Copyright 2026, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
;;
;; Tests that memory.fill only uses the low 8 bits of the value argument.
;; Calls memory.fill with value 0x100 (low 8 bits = 0x00), so all filled bytes must be 0x00.

(module
    (func (export "run")
        ;; fill 10 bytes at offset 0xaaa with value 0x100 - important to fill space
        ;; of length non-divisible by 8 to test if the final partial chunk is filled correctly
        ;;
        ;; only the low 8 bits (0x00) should be used, so memory must be all zeros
        (memory.fill (i32.const 0xaaa) (i32.const 0x100) (i32.const 10)))
    (func (export "run_nonzero")
        ;; fill 10 bytes at offset 0xbbb with value 0x1ab (low 8 bits = 0xab)
        ;; verifies the fix preserves non-zero low bits, not just zero-fills everything
        (memory.fill (i32.const 0xbbb) (i32.const 0x1ab) (i32.const 10)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0))
    (memory (export "memory") 1 1))
