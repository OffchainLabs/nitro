;; native_keccak256(input_ptr, input_len, output_ptr)
;;
;; Calldata layout : [0..N]    arbitrary input bytes
;; Return data     : [0..32]   keccak256(input)
;;
;; Variants for fixture authors (.json):
;;   * 0-byte input  (gas baseline)
;;   * 32-byte input (single absorption block)
;;   * 256-byte input
;;   * 2048-byte input (drives KECCAK_WORD_INK pricing)
;;
;; Pricing: input-size-dependent ink + the constant base. The fixture should
;; capture both ink_used and gas_used to keep per-byte cost regressions visible.
;;
;; The hostio name is `native_keccak256` (matches Nitro's vm_hooks export, even
;; though the brief calls it "keccak").

(module
    (import "vm_hooks" "native_keccak256"
        (func $keccak (param i32 i32 i32)))

    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))

    (memory (export "memory") 1)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; read calldata at offset 0
        (call $read_args (i32.const 0))

        ;; hash $args_len bytes starting at 0; write digest at offset 0x100
        (call $keccak (i32.const 0) (local.get $args_len) (i32.const 0x100))

        ;; return the 32-byte digest
        (call $write_result (i32.const 0x100) (i32.const 32))

        i32.const 0
    )
)
